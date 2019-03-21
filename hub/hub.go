// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hub

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/gorilla/websocket"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	Receive chan ReceivedData

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client

	RPCBuffer map[*[]byte]bool

	// Hub error logger.
	log *log.Logger
}

// NewHub is Hub constructed.
func NewHub() *Hub {
	return &Hub{
		Receive:    make(chan ReceivedData),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		RPCBuffer:  make(map[*[]byte]bool),
		// TODO using global logger
		log: log.New(os.Stderr, "iguagile-engine", log.Lshortfile),
	}
}

// RPC Targets
const (
	allClients = 0
	//otherClients         = 1
	allClientsBuffered   = 2
	otherClientsBuffered = 3
)

// Message types
const (
	//transform = 0
	//rpc = 1
	openMessage  = 2
	closeMessage = 3
)

// Run is provides backend synchronize goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			notify(h, client, openMessage)
			h.clients[client] = true
			for message := range h.RPCBuffer {
				client.Send <- *message
			}
		case client := <-h.Unregister:
			if _, ok := h.clients[client]; ok {
				closeConnection(h, client)
			}
		case receivedData := <-h.Receive:
			target := receivedData.Message[0]
			message := append(receivedData.Sender.ID, receivedData.Message[1:]...)
			for client := range h.clients {
				if client != receivedData.Sender || target == allClients || target == allClientsBuffered {
					select {
					case client.Send <- message:
					default:
						closeConnection(h, client)
					}
				}
			}
			if target == allClientsBuffered || target == otherClientsBuffered {
				receivedData.Sender.RPCBuffer[&message] = true
				h.RPCBuffer[&message] = true
			}
		}
	}
}

func notify(h *Hub, c *Client, messageType byte) {
	for client := range h.clients {
		message := append(c.ID, messageType)
		client.Send <- message
	}
}

func closeConnection(h *Hub, c *Client) {
	notify(h, c, closeMessage)
	for message := range c.RPCBuffer {
		delete(h.RPCBuffer, message)
	}
	delete(h.clients, c)
	close(c.Send)
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	Send chan []byte

	RPCBuffer map[*[]byte]bool

	ID []byte
}

// NewClient is Client constructor.
func NewClient(hub *Hub, conn *websocket.Conn) (*Client, error) {
	c := &Client{
		hub:       hub,
		conn:      conn,
		Send:      make(chan []byte, 256),
		RPCBuffer: make(map[*[]byte]bool),
	}

	uid, err := uuid.NewUUID()
	if err != nil {
		log.Fatal(err)
	}
	c.ID, err = uid.MarshalBinary()

	return c, err
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister <- c
		if err := c.conn.Close(); err != nil {
			c.hub.log.Println(err)
		}
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		c.hub.log.Println(err)
	}

	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			c.hub.log.Printf("error: %v", err)
		}
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.hub.Receive <- ReceivedData{Sender: c, Message: message}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err := c.conn.Close(); err != nil {
			c.hub.log.Println(err)
		}
	}()
	for {
		select {
		case message, ok := <-c.Send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.hub.log.Println(err)
			}
			if !ok {
				// The hub closed the channel.
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					c.hub.log.Println(err)
				}

				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			if _, err := w.Write(message); err != nil {
				c.hub.log.Println(err)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.hub.log.Println(err)
			}

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWs handles websocket requests from the peer.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client, err := NewClient(hub, conn)
	if err != nil {
		log.Println(err)
		return
	}
	client.hub.Register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

// ReceivedData is Client side transfer data struct.
type ReceivedData struct {
	Sender  *Client
	Message []byte
}
