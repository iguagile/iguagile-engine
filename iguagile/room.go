package iguagile

import (
	"encoding/binary"
	"log"
	"math"
	"os"

	"github.com/iguagile/iguagile-engine/data"
	"github.com/iguagile/iguagile-engine/id"
)

// Room maintains the set of active clients and broadcasts messages to the
// clients.
type Room struct {
	id               int
	clientManager    *ClientManager
	objectManager    *GameObjectManager
	rpcBufferManager *RPCBufferManager
	generator        *id.Generator
	log              *log.Logger
	host             *Client
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
		id:               roomID,
		clientManager:    NewClientManager(),
		objectManager:    NewGameObjectManager(),
		rpcBufferManager: NewRPCBufferManager(),
		generator:        gen,
		log:              log.New(os.Stdout, "iguagile-engine ", log.Lshortfile),
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
	r.objectManager.Lock()
	defer r.objectManager.Unlock()

	go client.writeStart()
	client.Send(append(client.GetIDByte(), register))
	message := append(client.GetIDByte(), newConnection)
	r.SendToOtherClients(message, client)
	if err := r.clientManager.Add(client); err != nil {
		r.log.Println(err)
		return
	}

	r.rpcBufferManager.SendRPCBuffer(client)
	r.rpcBufferManager.Add(message, client)

	for _, obj := range r.objectManager.GetAllGameObjects() {
		objectIDByte := make([]byte, 4)
		binary.LittleEndian.PutUint32(objectIDByte, uint32(obj.id))
		payload := append(objectIDByte, obj.resourcePath...)
		msg := append(append(obj.owner.GetIDByte(), instantiate), payload...)
		client.Send(msg)
	}

	if r.clientManager.Count() == 1 {
		r.host = client
		message := append(client.GetIDByte(), migrateHost)
		client.Send(message)
	}

	go client.readStart()
}

// Unregister requests from clients.
func (r *Room) Unregister(client *Client) {
	r.generator.Free(client.GetID())
	r.clientManager.Remove(client.GetID())
	r.rpcBufferManager.Remove(client)

	if r.clientManager.Count() == 0 {
		r.objectManager.Clear()
		return
	}

	if client == r.host {
		c, err := r.clientManager.First()
		if err != nil {
			r.log.Println(err)
		} else {
			r.host = c
			message := append(c.GetIDByte(), migrateHost)
			c.Send(message)
		}
	}

	for _, obj := range r.objectManager.GetAllGameObjects() {
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
		r.rpcBufferManager.Add(message, sender)
	case AllClientsBuffered:
		r.SendToAllClients(message)
		r.rpcBufferManager.Add(message, sender)
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

	if ok := r.objectManager.Exist(objID); ok {
		return
	}

	obj := &GameObject{
		owner:        sender,
		id:           objID,
		lifetime:     data[4],
		resourcePath: resourcePath,
	}
	if err := r.objectManager.Add(obj); err != nil {
		r.log.Println(err)
		return
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

	obj, err := r.objectManager.Get(objID)
	if err != nil {
		r.log.Println(err)
		return
	}

	if obj.owner != sender {
		return
	}

	r.objectManager.Remove(objID)

	message := append(append(sender.GetIDByte(), destroy), idByte...)
	r.SendToAllClients(message)
}

func (r *Room) destroyObject(gameObject *GameObject) {
	r.objectManager.Remove(gameObject.id)
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
	obj, err := r.objectManager.Get(objID)
	if err != nil {
		r.log.Println(err)
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

	if !r.clientManager.Exist(clientID) {
		return
	}

	obj, err := r.objectManager.Get(objID)
	if err != nil {
		r.log.Println(err)
		return
	}

	if obj.owner != sender {
		return
	}

	client, err := r.clientManager.Get(clientID)
	if err != nil {
		r.log.Println(err)
		return
	}

	message := append(append(sender.GetIDByte(), transferObjectControlAuthority), objIDByte...)
	client.Send(message)
	obj.owner = client
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
}

// SendToAllClients sends outbound message to all registered clients.
func (r *Room) SendToAllClients(message []byte) {
	r.clientManager.Lock()
	defer r.clientManager.Unlock()
	for _, client := range r.clientManager.GetAllClients() {
		client.Send(message)
	}
}

// SendToOtherClients sends outbound message to other registered clients.
func (r *Room) SendToOtherClients(message []byte, sender *Client) {
	r.clientManager.Lock()
	defer r.clientManager.Unlock()
	for _, client := range r.clientManager.GetAllClients() {
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
	r.clientManager.Lock()
	defer r.clientManager.Unlock()
	for _, client := range r.clientManager.GetAllClients() {
		if err := client.Close(); err != nil {
			r.log.Println(err)
		}
	}

	return nil
}
