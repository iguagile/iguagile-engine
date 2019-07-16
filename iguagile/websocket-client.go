package iguagile

import (
	"encoding/binary"

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
	idByte := make([]byte, 2)
	binary.LittleEndian.PutUint16(idByte, uint16(id))

	client := &ClientWebsocket{
		id:     id,
		idByte: idByte,
		conn:   conn,
		room:   room,
		send:   make(chan []byte),
	}

	return client, err
}

// Run is provides backend synchronize goroutine.
func (c *ClientWebsocket) Run() {
	go func() {
		for {
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				c.room.log.Println(err)
				c.room.CloseConnection(c)
				break
			}

			if err := c.room.Receive(c, message); err != nil {
				c.room.log.Println(err)
				c.room.CloseConnection(c)
			}

		}
	}()

	go func() {
		for {
			message := <-c.send
			if err := c.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				c.room.log.Println(err)
				c.room.CloseConnection(c)
				break
			}
		}
	}()
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

// Close closes the connection.
func (c *ClientWebsocket) Close() error {
	return c.conn.Close()
}
