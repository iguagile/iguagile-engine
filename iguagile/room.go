package iguagile

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"

	pb "github.com/iguagile/iguagile-room-proto/room"
)

// Room maintains the set of active clients and broadcasts messages to the
// clients.
type Room struct {
	clientManager    *ClientManager
	generator        *idGenerator
	log              *log.Logger
	host             *Client
	config           *RoomConfig
	creatorConnected bool
	roomProto        *pb.Room
	store            Store
	engine           *Engine
	service          RoomService
	streams          map[string]*Stream
	ready            bool
}

// RoomConfig is room config.
type RoomConfig struct {
	RoomID          int
	ApplicationName string
	Version         string
	Password        string
	MaxUser         int
	Info            map[string]string
	Token           []byte
}

func newRoom(engine *Engine, config *RoomConfig) (*Room, error) {
	gen, err := newIDGenerator()
	if err != nil {
		return nil, err
	}

	return &Room{
		clientManager: NewClientManager(),
		generator:     gen,
		log:           log.New(os.Stdout, "iguagile-engine ", log.Lshortfile),
		config:        config,
		store:         engine.store,
		roomProto:     &pb.Room{},
		engine:        engine,
		streams:       map[string]*Stream{},
	}, nil
}

func (r *Room) serve(ctx context.Context, conn *quicConn) error {
	client, err := NewClient(r, conn)
	if err != nil {
		return err
	}

	r.roomProto.ConnectedUser = int32(r.clientManager.count + 1)
	if err := r.store.RegisterRoom(r.roomProto); err != nil {
		r.log.Println(err)
	}
	return r.register(ctx, client)
}

const (
	// Maximum message size allowed from peer.
	maxMessageSize = math.MaxUint16
)

// register requests from the clients.
func (r *Room) register(ctx context.Context, client *Client) error {
	if err := r.clientManager.Add(client); err != nil {
		return err
	}

	if r.clientManager.Count() == 1 {
		r.host = client
	}

	for name, s := range r.streams {
		stream, err := client.conn.OpenStream()
		if err != nil {
			return err
		}

		if _, err := stream.Write([]byte(name)); err != nil {
			return err
		}

		s.streams[client.id] = stream
		client.streams[name] = stream
	}

	client.readStart(ctx)

	return r.service.OnRegisterClient(client.id)
}

// unregister requests from clients.
func (r *Room) unregister(client *Client) error {
	if err := r.generator.free(client.GetID()); err != nil {
		r.log.Println(err)
	}

	r.clientManager.Remove(client.GetID())
	if client == r.host {
		c, err := r.clientManager.First()
		if err != nil {
			return err
		}
		r.host = c
	}

	return r.service.OnUnregisterClient(client.id)
}

// CreateStream creates new stream.
// Call in the Create method implemented in RoomServiceFactory.
func (r *Room) CreateStream(streamName string) (*Stream, error) {
	if r.ready {
		return nil, errors.New("call CreateStream in the Create method implemented in RoomServiceFactory")
	}

	streamName = userStreamPrefix + streamName

	if _, ok := r.streams[streamName]; ok {
		return nil, fmt.Errorf("%v has already been created", streamName)
	}

	stream := &Stream{
		r:       r,
		streams: map[int]*quicStream{},
	}

	r.streams[streamName] = stream

	return stream, nil
}

// CloseConnection closes the connection and unregisters the client.
func (r *Room) CloseConnection(client *Client) {
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

	r.engine.rooms.Delete(r.config.RoomID)
	return r.service.Destroy()
}
