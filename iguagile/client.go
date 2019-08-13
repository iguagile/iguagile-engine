package iguagile

import "encoding/binary"

// Client is a middleman between the connection and the room.
type Client struct {
	id     int
	idByte []byte
	conn   Conn
	room   *Room
	send   chan []byte
}

// NewClient is Client constructed.
func NewClient(room *Room, conn Conn) (*Client, error) {
	id, err := room.generator.Generate()
	if err != nil {
		return nil, err
	}

	idByte := make([]byte, 2)
	binary.LittleEndian.PutUint16(idByte, uint16(id))

	client := &Client{
		id:     id,
		idByte: idByte,
		conn:   conn,
		room:   room,
		send:   make(chan []byte),
	}

	return client, nil
}

func (c *Client) readStart() {
	for {
		message, err := c.conn.Read()
		if err != nil {
			c.room.log.Println(err)
			c.room.CloseConnection(c)
			break
		}

		if err = c.room.Receive(c, message); err != nil {
			c.room.log.Println(err)
			c.room.CloseConnection(c)
			break
		}
	}
}

func (c *Client) writeStart() {
	for {
		if err := c.conn.Write(<-c.send); err != nil {
			c.room.log.Println(err)
			c.room.CloseConnection(c)
			break
		}
	}
}

// GetID is getter for id.
func (c *Client) GetID() int {
	return c.id
}

// GetIDByte is getter for idByte.
func (c *Client) GetIDByte() []byte {
	return c.idByte
}

// Send is enqueue outbound messages.
func (c *Client) Send(message []byte) {
	c.send <- message
}

// Close closes the connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
