package iguagile

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os"

	pb "github.com/iguagile/iguagile-room-proto/room"
)

// Room maintains the set of active clients and broadcasts messages to the
// clients.
type Room struct {
	clientManager    *ClientManager
	rpcBufferManager *RPCBufferManager
	generator        *IDGenerator
	log              *log.Logger
	host             *Client
	config           *RoomConfig
	creatorConnected bool
	roomProto        *pb.Room
	store            Store
	server           *RoomServer
	service          RoomService
}

// RoomConfig is room config.
type RoomConfig struct {
	RoomID          int
	ApplicationName string
	Version         string
	Password        string
	MaxUser         int
	Token           []byte
}

func newRoom(server *RoomServer, config *RoomConfig) (*Room, error) {
	gen, err := NewIDGenerator()
	if err != nil {
		return nil, err
	}

	return &Room{
		clientManager:    NewClientManager(),
		rpcBufferManager: NewRPCBufferManager(),
		generator:        gen,
		log:              log.New(os.Stdout, "iguagile-engine ", log.Lshortfile),
		config:           config,
		store:            server.store,
		roomProto:        &pb.Room{},
		server:           server,
	}, nil
}

func (r *Room) serve(conn io.ReadWriteCloser) error {
	client, err := NewClient(r, conn)
	if err != nil {
		return err
	}

	r.roomProto.ConnectedUser = int32(r.clientManager.count + 1)
	if err := r.store.RegisterRoom(r.roomProto); err != nil {
		r.log.Println(err)
	}
	return r.register(client)
}

// RPC target
const (
	allClients = iota
	otherClients
	allClientsBuffered
	otherClientsBuffered
	host
	server
)

// Message type
const (
	newConnection = iota
	exitConnection
	migrateHost
	register
)

const (
	// Maximum message size allowed from peer.
	maxMessageSize = math.MaxUint16
)

// register requests from the clients.
func (r *Room) register(client *Client) error {
	go client.writeStart()
	client.Send(append(client.GetIDByte(), register))
	message := []byte{newConnection}
	r.SendToOtherClients(client.id, message)
	if err := r.clientManager.Add(client); err != nil {
		return err
	}

	r.rpcBufferManager.SendRPCBuffer(client)
	r.rpcBufferManager.Add(message, client)

	if r.clientManager.Count() == 1 {
		r.host = client
		message := append(client.GetIDByte(), migrateHost)
		client.Send(message)
	}

	go client.readStart()
	return r.service.OnRegisterClient(client.id)
}

// unregister requests from clients.
func (r *Room) unregister(client *Client) error {
	if err := r.generator.Free(client.GetID()); err != nil {
		r.log.Println(err)
	}

	r.clientManager.Remove(client.GetID())
	r.rpcBufferManager.Remove(client)

	if client == r.host {
		c, err := r.clientManager.First()
		if err != nil {
			return err
		}
		r.host = c
		message := append(c.GetIDByte(), migrateHost)
		c.Send(message)
	}

	return r.service.OnUnregisterClient(client.id)
}

// receive is receive inbound messages from the clients.
func (r *Room) receive(sender *Client, receivedData []byte) error {
	inbound, err := NewInBoundData(receivedData)
	if err != nil {
		return err
	}

	message := receivedData[1:]
	if len(message) >= 1<<16 {
		return fmt.Errorf("too long message")
	}

	switch inbound.Target {
	case otherClients:
		r.SendToOtherClients(sender.id, message)
	case allClients:
		r.SendToAllClients(sender.id, message)
	case otherClientsBuffered:
		r.SendToOtherClients(sender.id, message)
		r.rpcBufferManager.Add(message, sender)
	case allClientsBuffered:
		r.SendToAllClients(sender.id, message)
		r.rpcBufferManager.Add(message, sender)
	case host:
		r.SendToHost(sender.id, message)
	case server:
		return r.service.Receive(sender.id, message)
	default:
		r.log.Println(receivedData)
	}

	return nil
}

// MigrateHost migrates host to the client.
func (r *Room) MigrateHost(sender *Client, idByte []byte) {
	if len(idByte) != 4 {
		r.log.Println("invalid payload length")
		return
	}

	if sender != r.host {
		return
	}

	clientID := int(binary.LittleEndian.Uint32(idByte))

	client, err := r.clientManager.Get(clientID)
	if err != nil {
		r.log.Println(err)
		return
	}

	r.host = client
	message := append(client.GetIDByte(), migrateHost)
	client.Send(message)
	if err := r.service.OnChangeHost(client.id); err != nil {
		r.log.Println(err)
	}
}

// SendToHost sends outbound message to the host.
func (r *Room) SendToHost(senderID int, message []byte) {
	message = append(make([]byte, 2, 2+len(message)), message...)
	binary.LittleEndian.PutUint16(message, uint16(senderID))
	r.host.Send(message)
}

// SendToClient sends outbound message to the client.
func (r *Room) SendToClient(targetID, senderID int, message []byte) {
	message = append(make([]byte, 2, 2+len(message)), message...)
	binary.LittleEndian.PutUint16(message, uint16(senderID))
	client, err := r.clientManager.Get(targetID)
	if err != nil {
		r.log.Println(err)
		return
	}

	client.Send(message)
}

// SendToAllClients sends outbound message to all registered clients.
func (r *Room) SendToAllClients(senderID int, message []byte) {
	message = append(make([]byte, 2, 2+len(message)), message...)
	binary.LittleEndian.PutUint16(message, uint16(senderID))
	r.clientManager.Lock()
	defer r.clientManager.Unlock()
	for _, client := range r.clientManager.GetAllClients() {
		client.Send(message)
	}
}

// SendToOtherClients sends outbound message to other registered clients.
func (r *Room) SendToOtherClients(senderID int, message []byte) {
	r.clientManager.Lock()
	defer r.clientManager.Unlock()
	for id, client := range r.clientManager.GetAllClients() {
		if id != senderID {
			client.Send(message)
		}
	}
}

// CloseConnection closes the connection and unregisters the client.
func (r *Room) CloseConnection(client *Client) {
	message := append(client.GetIDByte(), exitConnection)
	r.SendToOtherClients(client.id, message)
	if err := r.unregister(client); err != nil {
		r.log.Println(err)
	}
	if err := client.Close(); err != nil && err.Error() != "use of closed network connection" {
		r.log.Println(err)
	}
}

// Close closes all client connections.
func (r *Room) Close() error {
	r.clientManager.Lock()
	defer r.clientManager.Unlock()
	for _, client := range r.clientManager.GetAllClients() {
		if err := client.Close(); err != nil {
			r.log.Println(err)
		}
	}

	r.server.rooms.Delete(r.config.RoomID)
	return r.service.Destroy()
}
