package iguagile

import (
	"log"
	"os"
	"time"

	"github.com/iguagile/iguagile-engine/data"
)

// Room maintains the set of active clients and broadcasts messages to the
// clients.
type Room struct {
	id      int
	clients map[Client]bool
	buffer  map[*[]byte]Client
	log     *log.Logger
}

// NewRoom is Room constructed.
func NewRoom(serverID int, store Store) *Room {
	roomID, err := store.GenerateRoomID(serverID)
	if err != nil {
		log.Fatal(err)
	}

	return &Room{
		id:      roomID,
		clients: make(map[Client]bool),
		buffer:  make(map[*[]byte]Client),
		log:     log.New(os.Stderr, "iguagile-engine ", log.Lshortfile),
	}
}

// RPC target
const (
	AllClients = iota
	OtherClients
	AllClientsBuffered
	OtherClientsBuffered
)

// Message type
const (
	newConnection = iota
	exitConnection
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Register requests from the clients.
func (r *Room) Register(client Client) {
	go client.Run()
	message := append(client.GetID(), newConnection)
	client.SendToOtherClients(message)
	r.clients[client] = true
	for msg := range r.buffer {
		client.Send(*msg)
	}
	r.buffer[&message] = client
}

// Unregister requests from clients.
func (r *Room) Unregister(client Client) {
	for message, c := range r.buffer {
		if c == client {
			delete(r.buffer, message)
		}
	}
	delete(r.clients, client)
}

// Receive is receive inbound messages from the clients.
func (r *Room) Receive(sender Client, receivedData []byte) {
	rowData, err := data.NewBinaryData(receivedData, data.Inbound)
	if err != nil {
		r.log.Println(err)
	}
	message := append(append(sender.GetID(), rowData.MessageType), rowData.Payload...)
	if len(message) >= 1<<16 {
		r.log.Println("too long message")
		return
	}
	switch rowData.Target {
	case OtherClients:
		sender.SendToOtherClients(message)
	case AllClients:
		sender.SendToAllClients(message)
	case OtherClientsBuffered:
		sender.SendToOtherClients(message)
		r.buffer[&message] = sender
	case AllClientsBuffered:
		sender.SendToAllClients(message)
		r.buffer[&message] = sender
	default:
		r.log.Println(receivedData)
	}
}
