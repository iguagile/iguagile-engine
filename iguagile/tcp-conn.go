package iguagile

import (
	"encoding/binary"
	"net"
)

// ConnTCP is a wrapper of tcp connection.
type ConnTCP struct {
	conn *net.TCPConn
}

// Write writes a message.
func (c *ConnTCP) Write(message []byte) error {
	size := len(message)
	sizeByte := make([]byte, 2, size+2)
	binary.LittleEndian.PutUint16(sizeByte, uint16(size))
	message = append(sizeByte, message...)
	if _, err := c.conn.Write(message); err != nil {
		return err
	}
	return nil
}

// Read reads a message.
func (c *ConnTCP) Read() ([]byte, error) {
	buf := make([]byte, maxMessageSize)
	sizeBuf := make([]byte, 2)
	_, err := c.conn.Read(sizeBuf)
	if err != nil {
		return nil, err
	}

	size := int(binary.LittleEndian.Uint16(sizeBuf))
	receivedSizeSum := 0
	message := make([]byte, 0)
	for receivedSizeSum < size {
		receivedSize, err := c.conn.Read(buf[:size-receivedSizeSum])
		if err != nil {
			return nil, err
		}

		message = append(message, buf[:receivedSize]...)
		receivedSizeSum += receivedSize
	}

	return message, nil
}

// Close closes the connection.
func (c *ConnTCP) Close() error {
	return c.conn.Close()
}
