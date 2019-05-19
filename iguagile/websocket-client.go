package iguagile

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ClientWebsocket is a middleman between the websocket connection and the room.
type ClientWebsocket struct {
	id     []byte
	conn   *websocket.Conn
	room   *Room
	buffer []*[]byte
	send   chan []byte
}

// NewClientWebsocket is ClientWebsocket constructed.
func NewClientWebsocket(room *Room, conn *websocket.Conn) *ClientWebsocket {
	uid, err := uuid.NewUUID()
	if err != nil {
		log.Println(err)
	}
	return &ClientWebsocket{
		id:   uid[:],
		conn: conn,
		room: room,
		send: make(chan []byte),
	}
}

// Run is provides backend synchronize goroutine.
func (c *ClientWebsocket) Run() {
	go c.readPump()
	go c.writePump()
}

// GetID is getter for id
func (c *ClientWebsocket) GetID() []byte {
	return c.id
}

// Send is enqueue outbound messages
func (c *ClientWebsocket) Send(message []byte) {
	c.send <- message
}

// SendToAllClients is send outbound message to all registered clients
func (c *ClientWebsocket) SendToAllClients(message []byte) {
	for client := range c.room.clients {
		client.Send(message)
	}
}

// SendToOtherClients is send outbound message to other registered clients
func (c *ClientWebsocket) SendToOtherClients(message []byte) {
	for client := range c.room.clients {
		if client != c {
			client.Send(message)
		}
	}
}

// CloseConnection is disconnect and unregister client
func (c *ClientWebsocket) CloseConnection() {
	message := append(c.id, exitConnection)
	c.SendToOtherClients(message)
	for _, message := range c.buffer {
		delete(c.room.buffer, message)
	}
	delete(c.room.clients, c)
	if err := c.conn.Close(); err != nil {
		c.room.log.Println(err)
	}
}

// AddBuffer is buffer messages
func (c *ClientWebsocket) AddBuffer(message *[]byte) {
	c.buffer = append(c.buffer, message)
	c.room.buffer[message] = true
}

// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// https://github.com/gorilla/websocket/blob/master/LICENSE

func (c *ClientWebsocket) readPump() {
	defer func() {
		c.CloseConnection()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		c.room.log.Println(err)
	}

	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			c.room.log.Printf("error: %v", err)
		}
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.room.log.Printf("error: %v", err)
			}
			break
		}

		c.room.Receive(c, message)
	}
}

func (c *ClientWebsocket) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.CloseConnection()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.room.log.Println(err)
			}
			if !ok {
				// The room closed the channel.
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					c.room.log.Println(err)
				}

				return
			}

			w, err := c.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				c.room.log.Println(err)
				return
			}

			if _, err := w.Write(message); err != nil {
				c.room.log.Println(err)
			}

			if err := w.Close(); err != nil {
				c.room.log.Println(err)
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.room.log.Println(err)
			}

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
