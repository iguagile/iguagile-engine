// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hub

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/iguagile/iguagile-engine/data"

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
	allClients = iota
	otherClients
	allClientsBuffered
	otherClientsBuffered
)

// Message type
const (
	newConnection = iota
	exitConnection
)

// Run is provides backend synchronize goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			client.notify(newConnection)
			h.clients[client] = true
			for message := range h.RPCBuffer {
				client.Send <- *message
			}
		case client := <-h.Unregister:
			if _, ok := h.clients[client]; ok {
				client.closeConnection()
			}
		case receivedData := <-h.Receive:
			rowData, err := data.NewBinaryData(receivedData.Message, data.Inbound)
			if err != nil {
				log.Println(err)
			}

			sender := receivedData.Sender
			message := append(append(sender.ID, rowData.MessageType), rowData.Payload...)
			switch rowData.Target {
			case otherClients:
				sender.sendToOtherClients(message)
			case allClients:
				sender.sendToAllClients(message)
			case otherClientsBuffered:
				sender.sendToOtherClients(message)
				sender.addBuffer(&message)
			case allClientsBuffered:
				sender.sendToAllClients(message)
				sender.addBuffer(&message)
			}
		}
	}
}

func (c *Client) addBuffer(message *[]byte) {
	c.RPCBuffer[message] = true
	c.hub.RPCBuffer[message] = true
}

func (c *Client) sendToAllClients(message []byte) {
	for client := range c.hub.clients {
		select {
		case client.Send <- message:
		default:
			c.closeConnection()
		}
	}
}

func (c *Client) sendToOtherClients(message []byte) {
	for client := range c.hub.clients {
		if client != c {
			select {
			case client.Send <- message:
			default:
				client.closeConnection()
			}
		}
	}
}

func (c *Client) notify(messageType byte) {
	message := append(c.ID, messageType)
	c.sendToOtherClients(message)
	c.addBuffer(&message)
}

func (c *Client) closeConnection() {
	c.notify(exitConnection)
	// TODO subType add RPCBuffer.
	for message := range c.RPCBuffer {
		delete(c.hub.RPCBuffer, message)
	}
	delete(c.hub.clients, c)
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
		log.Println(err)
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

			w, err := c.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				c.hub.log.Println(err)
				return
			}

			if _, err := w.Write(message); err != nil {
				c.hub.log.Println(err)
			}

			if err := w.Close(); err != nil {
				c.hub.log.Println(err)
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
