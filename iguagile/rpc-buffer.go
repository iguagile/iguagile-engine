package iguagile

import (
	"sync"
)

// RPCBufferManager manages buffered rpc messages.
type RPCBufferManager struct {
	buffer map[*[]byte]*Client
	sync.Mutex
}

// NewRPCBufferManager is RPCBufferManger constructed.
func NewRPCBufferManager() *RPCBufferManager {
	return &RPCBufferManager{
		buffer: make(map[*[]byte]*Client),
		Mutex:  sync.Mutex{},
	}
}

// Add new rpc message.
func (m *RPCBufferManager) Add(message []byte, sender *Client) {
	m.Lock()
	m.buffer[&message] = sender
	m.Unlock()
}

// Remove rpc messages.
func (m *RPCBufferManager) Remove(client *Client) {
	m.Lock()
	for buffer, sender := range m.buffer {
		if sender == client {
			delete(m.buffer, buffer)
		}
	}
	m.Unlock()
}

// Clear all rpc messages.
func (m *RPCBufferManager) Clear() {
	m.Lock()
	m.buffer = make(map[*[]byte]*Client)
	m.Unlock()
}

// SendRPCBuffer sends all buffered rpc messages.
func (m *RPCBufferManager) SendRPCBuffer(client *Client) {
	m.Lock()
	for buffer := range m.buffer {
		client.Send(*buffer)
	}
	m.Unlock()
}
