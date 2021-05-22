package iguagile

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
)

// Client is a middleman between the connection and the room.
type Client struct {
	id      int
	idByte  []byte
	conn    *quicConn
	streams map[string]*quicStream
	room    *Room
}

// NewClient is Client constructed.
func NewClient(room *Room, conn *quicConn) (*Client, error) {
	id, err := room.generator.generate()
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
	}

	return client, nil
}

func (c *Client) readStart() {
	for {
		buf := make([]byte, maxMessageSize)
		stream, err := c.conn.AcceptStream()
		if err != nil {
			c.room.log.Println(err)
			c.room.CloseConnection(c)
			break
		}

		go func() {
			n, err := stream.Read(buf)
			if err != nil {
				c.room.log.Println(err)
			}

			streamName := string(buf[:n])
			c.streams[streamName] = stream
			receive, err := c.room.service.ReceiveFunc(streamName)
			if err != nil {
				c.room.log.Println(err)
			}

			for {
				n, err := stream.Read(buf)
				if err != nil {
					c.room.log.Println(err)
					break
				}

				if err := receive(c.id, buf[:n]); err != nil {
					c.room.log.Println(err)
				}
			}
		}()
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

// Close closes the connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// ClientManager manages clients.
type ClientManager struct {
	clients map[int]*Client
	count   int
	*sync.Mutex
}

// NewClientManager is ClientManager constructed.
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[int]*Client),
		Mutex:   &sync.Mutex{},
	}
}

// Get the client.
func (m *ClientManager) Get(clientID int) (*Client, error) {
	m.Lock()
	client, ok := m.clients[clientID]
	m.Unlock()
	if !ok {
		return nil, fmt.Errorf("client not exists %v", clientID)
	}

	return client, nil
}

// Add the client.
func (m *ClientManager) Add(client *Client) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client.GetID()]; ok {
		return fmt.Errorf("client already exists %v", client.GetID())
	}

	m.clients[client.GetID()] = client
	m.count++
	return nil
}

// Remove the client.
func (m *ClientManager) Remove(clientID int) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[clientID]; !ok {
		return
	}

	delete(m.clients, clientID)
	m.count--
}

// Exist checks the client exists.
func (m *ClientManager) Exist(clientID int) bool {
	m.Lock()
	_, ok := m.clients[clientID]
	m.Unlock()
	return ok
}

// GetAllClients returns all clients.
func (m *ClientManager) GetAllClients() map[int]*Client {
	return m.clients
}

// Clear all clients.
func (m *ClientManager) Clear() {
	m.Lock()
	m.clients = make(map[int]*Client)
	m.Unlock()
}

// Count clients.
func (m *ClientManager) Count() int {
	return m.count
}

// First returns a first element.
func (m *ClientManager) First() (*Client, error) {
	m.Lock()
	defer m.Unlock()

	for _, client := range m.clients {
		return client, nil
	}

	return nil, errors.New("clients not exist")
}
