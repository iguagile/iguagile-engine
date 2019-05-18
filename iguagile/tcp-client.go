package iguagile

import (
	"github.com/google/uuid"
	"log"
	"net"
)

type ClientTCP struct {
	id     []byte
	conn   *net.TCPConn
	room   *Room
	buffer map[*[]byte]bool
	send   chan []byte
}

func NewClientTCP(room *Room, conn *net.TCPConn) *ClientTCP {
	uid, err := uuid.NewUUID()
	if err != nil {
		log.Println(err)
	}
	return &ClientTCP{
		id:     uid[:],
		conn:   conn,
		room:   room,
		buffer: make(map[*[]byte]bool),
		send:   make(chan []byte),
	}
}

func (c *ClientTCP) Run() {
	go func() {
		msgSize := make([]byte, 1)
		buf := make([]byte, maxMessageSize)
		for {
			_, err := c.conn.Read(msgSize)
			if err != nil {
				c.room.log.Println(err)
				c.CloseConnection()
				break
			}
			n, err := c.conn.Read(buf[:msgSize[0]])
			if err != nil {
				c.room.log.Println(err)
				c.CloseConnection()
				break
			}
			if byte(n) != msgSize[0] {
				c.CloseConnection()
				break
			}
			c.room.Receive(c, buf[:n])
		}
	}()
	go func() {
		for {
			message := <-c.send
			message = append([]byte{byte(len(message))}, message...)
			_, err := c.conn.Write(message)
			if err != nil {
				c.room.log.Println(err)
				c.CloseConnection()
				break
			}
		}
	}()
}

func (c *ClientTCP) GetID() []byte {
	return c.id
}

func (c *ClientTCP) Send(message []byte) {
	c.send <- message
}

func (c *ClientTCP) SendToAllClients(message []byte) {
	for client := range c.room.clients {
		client.Send(message)
	}
}

func (c *ClientTCP) SendToOtherClients(message []byte) {
	for client := range c.room.clients {
		if client != c {
			client.Send(message)
		}
	}
}

func (c *ClientTCP) CloseConnection() {
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

func (c *ClientTCP) AddBuffer(message *[]byte) {
	c.buffer[message] = true
	c.room.buffer[message] = true
}
