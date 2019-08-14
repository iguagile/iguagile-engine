package iguagile

import (
	"sync"
)

type RPCBufferManager struct {
	buffer map[*[]byte]*Client
	sync.Mutex
}

func NewRPCBufferManager() *RPCBufferManager {
	return &RPCBufferManager{
		buffer: make(map[*[]byte]*Client),
		Mutex:  sync.Mutex{},
	}
}

func (m *RPCBufferManager) Add(message []byte, sender *Client) {
	m.Lock()
	m.buffer[&message] = sender
	m.Unlock()
}

func (m *RPCBufferManager) Remove(client *Client) {
	m.Lock()
	for buffer, sender := range m.buffer {
		if sender == client {
			delete(m.buffer, buffer)
		}
	}
	m.Unlock()
}

func (m *RPCBufferManager) Clear() {
	m.Lock()
	m.buffer = make(map[*[]byte]*Client)
	m.Unlock()
}

func (m *RPCBufferManager) SendRPCBuffer(client *Client) {
	m.Lock()
	for buffer := range m.buffer {
		client.Send(*buffer)
	}
	m.Unlock()
}
