package iguagile

import (
	"encoding/binary"
	"fmt"
	"net"
)

// ClientTCP is a middleman between the tcp connection and the room.
type ClientTCP struct {
	id     int
	idByte []byte
	conn   *net.TCPConn
	room   *Room
	send   chan []byte
}

// NewClientTCP is ClientTCP constructed.
func NewClientTCP(room *Room, conn *net.TCPConn) (*ClientTCP, error) {
	id, err := room.generator.Generate()
	idByte := make([]byte, 2)
	binary.LittleEndian.PutUint16(idByte, uint16(id))

	client := &ClientTCP{
		id:     id,
		idByte: idByte,
		conn:   conn,
		room:   room,
		send:   make(chan []byte),
	}

	return client, err
}

// Run is provides backend synchronize goroutine.
func (c *ClientTCP) Run() {
	go func() {
		sizeBuf := make([]byte, 2)
		buf := make([]byte, maxMessageSize)
		for {
			_, err := c.conn.Read(sizeBuf)
			if err != nil {
				c.room.log.Println(err)
				c.room.CloseConnection(c)
				break
			}
			size := int(binary.LittleEndian.Uint16(sizeBuf))
			n, err := c.conn.Read(buf[:size])
			if err != nil {
				c.room.log.Println(err)
				c.room.CloseConnection(c)
				break
			}
			if n != size {
				msg := fmt.Sprintf("data size does not match. ClientID:%v %vbyte %vbyte", c.id, size, n)
				c.room.log.Println(msg)
				c.room.CloseConnection(c)
				break
			}

			if err = c.room.Receive(c, buf[:n]); err != nil {
				c.room.log.Println(err)
				c.room.CloseConnection(c)
				break
			}

		}
	}()
	go func() {
		for {
			message := <-c.send
			size := len(message)
			sizeByte := make([]byte, 2, size+2)
			binary.LittleEndian.PutUint16(sizeByte, uint16(size))
			message = append(sizeByte, message...)
			_, err := c.conn.Write(message)
			if err != nil {
				c.room.log.Println(err)
				c.room.CloseConnection(c)
				break
			}
		}
	}()
}

// GetID is getter for id.
func (c *ClientTCP) GetID() int {
	return c.id
}

// GetIDByte is getter for idByte.
func (c *ClientTCP) GetIDByte() []byte {
	return c.idByte
}

// Send is enqueue outbound messages.
func (c *ClientTCP) Send(message []byte) {
	c.send <- message
}

// Close closes the connection.
func (c *ClientTCP) Close() error {
	return c.conn.Close()
}
