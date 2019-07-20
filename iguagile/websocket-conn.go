package iguagile

import (
	"github.com/gorilla/websocket"
)

// ConnWebsocket is a wrapper of websocket connection.
type ConnWebsocket struct {
	conn *websocket.Conn
}

// Write writes a message.
func (c *ConnWebsocket) Write(message []byte) error {
	return c.conn.WriteMessage(websocket.BinaryMessage, message)
}

// Read reads a message.
func (c *ConnWebsocket) Read() ([]byte, error) {
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	return message, nil
}

// Close closes the connection.
func (c *ConnWebsocket) Close() error {
	return c.conn.Close()
}
