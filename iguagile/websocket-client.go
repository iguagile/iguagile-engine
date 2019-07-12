package iguagile

import (
	"time"

	"github.com/gorilla/websocket"
)

// ClientWebsocket is a middleman between the websocket connection and the room.
type ClientWebsocket struct {
	id     int
	idByte []byte
	conn   *websocket.Conn
	room   *Room
	send   chan []byte
}

// NewClientWebsocket is ClientWebsocket constructed.
func NewClientWebsocket(room *Room, conn *websocket.Conn) (*ClientWebsocket, error) {
	id, err := room.generator.Generate()

	client := &ClientWebsocket{
		id:     id,
		idByte: make([]byte, 2),
		conn:   conn,
		room:   room,
		send:   make(chan []byte),
	}
	client.idByte[0] = byte(id & 0xff)
	client.idByte[1] = byte(id >> 8)

	return client, err
}

// Run is provides backend synchronize goroutine.
func (c *ClientWebsocket) Run() {
	go c.readPump()
	go c.writePump()
}

// GetID is getter for id
func (c *ClientWebsocket) GetID() int {
	return c.id
}

// GetIDByte is getter for idByte
func (c *ClientWebsocket) GetIDByte() []byte {
	return c.idByte
}

// Send is enqueue outbound messages
func (c *ClientWebsocket) Send(message []byte) {
	c.send <- message
}

func (c *ClientWebsocket) Close() error {
	return c.conn.Close()
}

// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// https://github.com/gorilla/websocket/blob/master/LICENSE

func (c *ClientWebsocket) readPump() {
	defer func() {
		c.room.CloseConnection(c)
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
		c.room.CloseConnection(c)
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
