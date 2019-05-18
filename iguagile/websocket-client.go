package iguagile

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

type ClientWebsocket struct {
	id     []byte
	conn   websocket.Conn
	room   *Room
	buffer map[*[]byte]bool
	send   chan []byte
}

func NewClientWebsocket(room *Room, conn websocket.Conn) *ClientWebsocket {
	uid, err := uuid.NewUUID()
	if err != nil {
		log.Println(err)
	}
	return &ClientWebsocket{
		id:     uid[:],
		conn:   conn,
		room:   room,
		buffer: make(map[*[]byte]bool),
		send:   make(chan []byte),
	}
}

func (c *ClientWebsocket) Run() {
	go c.readPump()
	go c.writePump()
}

func (c *ClientWebsocket) GetID() []byte {
	return c.id
}

func (c *ClientWebsocket) Send(message []byte) {
	c.send <- message
}

func (c *ClientWebsocket) SendToAllClients(message []byte) {
	for client := range c.room.clients {
		client.Send(message)
	}
}

func (c *ClientWebsocket) SendToOtherClients(message []byte) {
	for client := range c.room.clients {
		if client != c {
			client.Send(message)
		}
	}
}

func (c *ClientWebsocket) CloseConnection() {
	message := append(c.id, exitConnection)
	c.SendToOtherClients(message)
	for message := range c.buffer {
		delete(c.room.buffer, message)
	}
	delete(c.room.clients, c)
	if err := c.conn.Close(); err != nil {
		c.room.log.Println(err)
	}
}

func (c *ClientWebsocket) AddBuffer(message *[]byte) {
	c.buffer[message] = true
	c.room.buffer[message] = true
}

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
				// The hub closed the channel.
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
