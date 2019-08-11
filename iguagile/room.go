package iguagile

import (
	"encoding/binary"
	"log"
	"math"
	"os"
	"sync"

	"github.com/iguagile/iguagile-engine/data"
	"github.com/iguagile/iguagile-engine/id"
)

// Room maintains the set of active clients and broadcasts messages to the
// clients.
type Room struct {
	id          int
	clients     map[int]*Client
	clientsLock *sync.Mutex
	buffer      map[*[]byte]*Client
	objects     map[int]*GameObject
	objectsLock *sync.Mutex
	generator   *id.Generator
	log         *log.Logger
	host        *Client
}

// NewRoom is Room constructed.
func NewRoom(serverID int, store Store) *Room {
	roomID, err := store.GenerateRoomID(serverID)
	if err != nil {
		log.Fatal(err)
	}

	gen, err := id.NewGenerator(math.MaxInt16)
	if err != nil {
		log.Fatal(err)
	}

	return &Room{
		id:          roomID,
		clients:     make(map[int]*Client),
		clientsLock: &sync.Mutex{},
		buffer:      make(map[*[]byte]*Client),
		objects:     make(map[int]*GameObject),
		objectsLock: &sync.Mutex{},
		generator:   gen,
		log:         log.New(os.Stdout, "iguagile-engine ", log.Lshortfile),
	}
}

// RPC target
const (
	AllClients = iota
	OtherClients
	AllClientsBuffered
	OtherClientsBuffered
	Host
	Server
)

// Message type
const (
	newConnection = iota
	exitConnection
	instantiate
	destroy
	requestObjectControlAuthority
	transferObjectControlAuthority
	migrateHost
	register
	transform
	rpc
)

const (
	// Maximum message size allowed from peer.
	maxMessageSize = math.MaxUint16
)

// Register requests from the clients.
func (r *Room) Register(client *Client) {
	r.clientsLock.Lock()
	defer r.clientsLock.Unlock()

	go client.writeStart()
	client.Send(append(client.GetIDByte(), register))
	message := append(client.GetIDByte(), newConnection)
	r.SendToOtherClients(message, client)
	r.clients[client.GetID()] = client
	for msg := range r.buffer {
		client.Send(*msg)
	}
	r.buffer[&message] = client

	r.objectsLock.Lock()
	defer r.objectsLock.Unlock()
	for _, obj := range r.objects {
		objectIDByte := make([]byte, 4)
		binary.LittleEndian.PutUint32(objectIDByte, uint32(obj.id))
		payload := append(objectIDByte, obj.resourcePath...)
		msg := append(append(obj.owner.GetIDByte(), instantiate), payload...)
		client.Send(msg)
	}

	if len(r.clients) == 1 {
		r.host = client
		message := append(client.GetIDByte(), migrateHost)
		client.Send(message)
	}

	go client.readStart()
}

// Unregister requests from clients.
func (r *Room) Unregister(client *Client) {
	r.clientsLock.Lock()
	defer r.clientsLock.Unlock()

	cid := client.GetID()
	for message, c := range r.buffer {
		if c == client {
			delete(r.buffer, message)
		}
	}
	r.generator.Free(cid)
	delete(r.clients, client.GetID())

	r.objectsLock.Lock()
	defer r.objectsLock.Unlock()
	if len(r.clients) == 0 {
		r.objects = make(map[int]*GameObject)
		return
	}

	if client == r.host && len(r.clients) > 0 {
		for _, c := range r.clients {
			r.host = c
			message := append(c.GetIDByte(), migrateHost)
			c.Send(message)
			break
		}
	}

	for _, obj := range r.objects {
		if obj.owner == client {
			switch obj.lifetime {
			case roomExist:
				r.transferObjectControlAuthority(obj, r.host)
			case ownerExist:
				r.destroyObject(obj)
			}
		}
	}
}

// Receive is receive inbound messages from the clients.
func (r *Room) Receive(sender *Client, receivedData []byte) error {
	inbound, err := data.NewInBoundData(receivedData)
	if err != nil {
		return err
	}

	message := append(append(sender.GetIDByte(), inbound.MessageType), inbound.Payload...)
	if len(message) >= 1<<16 {
		r.log.Println("too long message")
		// TODO Decide how to use it in practice
		// return error or logging
		return nil
	}

	switch inbound.Target {
	case OtherClients:
		r.SendToOtherClients(message, sender)
	case AllClients:
		r.SendToAllClients(message)
	case OtherClientsBuffered:
		r.SendToOtherClients(message, sender)
		r.buffer[&message] = sender
	case AllClientsBuffered:
		r.SendToAllClients(message)
		r.buffer[&message] = sender
	case Host:
		r.host.Send(message)
	case Server:
		r.ReceiveRPC(sender, inbound)
	default:
		r.log.Println(receivedData)
	}

	return nil
}

// ReceiveRPC receives rpc to server.
func (r *Room) ReceiveRPC(sender *Client, binaryData *data.BinaryData) {
	switch binaryData.MessageType {
	case instantiate:
		r.InstantiateObject(sender, binaryData.Payload)
	case destroy:
		r.DestroyObject(sender, binaryData.Payload)
	case requestObjectControlAuthority:
		r.RequestObjectControlAuthority(sender, binaryData.Payload)
	case transferObjectControlAuthority:
		r.TransferObjectControlAuthority(sender, binaryData.Payload)
	case migrateHost:
		r.MigrateHost(sender, binaryData.Payload)
	default:
		r.log.Println(binaryData)
	}
}

// InstantiateObject instantiates the game object.
func (r *Room) InstantiateObject(sender *Client, data []byte) {
	if len(data) <= 4 {
		r.log.Println("invalid data length")
		return
	}

	objIDByte := data[:4]
	objID := int(binary.LittleEndian.Uint32(objIDByte))
	resourcePath := data[5:]

	r.objectsLock.Lock()
	defer r.objectsLock.Unlock()
	if _, ok := r.objects[objID]; ok {
		return
	}

	r.objects[objID] = &GameObject{
		owner:        sender,
		id:           objID,
		lifetime:     data[4],
		resourcePath: resourcePath,
	}

	message := append(append(append(sender.GetIDByte(), instantiate), objIDByte...), resourcePath...)
	r.SendToAllClients(message)
}

// DestroyObject destroys the game object.
func (r *Room) DestroyObject(sender *Client, idByte []byte) {
	if len(idByte) != 4 {
		r.log.Println("invalid object id")
		return
	}

	objID := int(binary.LittleEndian.Uint32(idByte))

	r.objectsLock.Lock()
	defer r.objectsLock.Unlock()
	obj, ok := r.objects[objID]
	if !ok {
		return
	}

	if obj.owner != sender {
		return
	}

	delete(r.objects, objID)

	message := append(append(sender.GetIDByte(), destroy), idByte...)
	r.SendToAllClients(message)
}

func (r *Room) destroyObject(gameObject *GameObject) {
	delete(r.objects, gameObject.id)
	idByte := make([]byte, 4)
	binary.LittleEndian.PutUint32(idByte, uint32(gameObject.id))
	message := append(append(gameObject.owner.GetIDByte(), destroy), idByte...)
	r.SendToAllClients(message)
}

// RequestObjectControlAuthority requests control authority of the object to the owner of the object.
func (r *Room) RequestObjectControlAuthority(sender *Client, idByte []byte) {
	if len(idByte) != 4 {
		r.log.Println("invalid payload length")
		return
	}

	objID := int(binary.LittleEndian.Uint32(idByte))
	obj, ok := r.objects[objID]
	if !ok {
		return
	}

	message := append(append(sender.GetIDByte(), requestObjectControlAuthority), idByte...)
	obj.owner.Send(message)
}

// TransferObjectControlAuthority transfers control authority of the object.
func (r *Room) TransferObjectControlAuthority(sender *Client, payload []byte) {
	if len(payload) != 8 {
		r.log.Println("invalid payload length")
		return
	}

	objIDByte := payload[:4]
	objID := int(binary.LittleEndian.Uint32(objIDByte))

	clientIDByte := payload[4:8]
	clientID := int(binary.LittleEndian.Uint32(clientIDByte))

	if _, ok := r.clients[clientID]; !ok {
		return
	}

	obj, ok := r.objects[objID]
	if !ok {
		return
	}

	if obj.owner != sender {
		return
	}

	message := append(append(sender.GetIDByte(), transferObjectControlAuthority), objIDByte...)
	for cid, client := range r.clients {
		if cid == clientID {
			client.Send(message)
			obj.owner = client
		}
	}
}

func (r *Room) transferObjectControlAuthority(gameObject *GameObject, client *Client) {
	idByte := make([]byte, 4)
	binary.LittleEndian.PutUint32(idByte, uint32(gameObject.id))
	message := append(append(gameObject.owner.GetIDByte(), transferObjectControlAuthority), idByte...)
	client.Send(message)
	gameObject.owner = client
}

// MigrateHost migrates host to the client.
func (r *Room) MigrateHost(sender *Client, idByte []byte) {
	if len(idByte) != 4 {
		r.log.Println("invalid payload length")
		return
	}

	clientID := int(binary.LittleEndian.Uint32(idByte))

	for cid, client := range r.clients {
		if cid == clientID {
			message := append(client.GetIDByte(), migrateHost)
			client.Send(message)
			break
		}
	}
}

// SendToAllClients sends outbound message to all registered clients.
func (r *Room) SendToAllClients(message []byte) {
	for _, client := range r.clients {
		client.Send(message)
	}
}

// SendToOtherClients sends outbound message to other registered clients.
func (r *Room) SendToOtherClients(message []byte, sender *Client) {
	for _, client := range r.clients {
		if client != sender {
			client.Send(message)
		}
	}
}

// CloseConnection closes the connection and unregisters the client.
func (r *Room) CloseConnection(client *Client) {
	message := append(client.GetIDByte(), exitConnection)
	r.SendToOtherClients(message, client)
	r.Unregister(client)
	if err := client.Close(); err != nil && err.Error() != "use of closed network connection" {
		r.log.Println(err)
	}
}

// Close closes all client connections.
func (r *Room) Close() error {
	for _, client := range r.clients {
		if err := client.Close(); err != nil {
			r.log.Println(err)
		}
	}

	return nil
}
