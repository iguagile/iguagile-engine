package iguagile

import (
	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	pb "github.com/iguagile/iguagile-room-proto/room"
)

// Store is an interface for connecting to backend storage and storing data.
type Store interface {
	Close() error
	GenerateServerID() (int, error)
	RegisterServer(*pb.Server) error
	UnregisterServer(*pb.Server) error
	RegisterRoom(*pb.Room) error
	UnregisterRoom(*pb.Room) error
}

// Redis is a structure that wraps goredis to make it easy to use.
type Redis struct {
	conn redis.Conn
}

const (
	registerServerMessage = iota
	unregisterServerMessage
	registerRoomMessage
	unregisterRoomMessage
)

// GenerateServerID is a method to number unique ServerID.
func (r *Redis) GenerateServerID() (int, error) {
	// TODO CHECK TO 1 << 16 over.
	i, err := redis.Int(r.conn.Do("INCR", "server_id"))
	return i << 16, err
}

// RegisterServer registers server to redis.
func (r *Redis) RegisterServer(server *pb.Server) error {
	serverProto, err := proto.Marshal(server)
	if err != nil {
		return err
	}

	message := append([]byte{registerServerMessage}, serverProto...)

	_, err = r.conn.Do("PUBLISH", "channel_servers", message)
	if err != nil {
		return err
	}

	return nil
}

// UnregisterServer unregisters server from redis.
func (r *Redis) UnregisterServer(server *pb.Server) error {
	serverProto, err := proto.Marshal(server)
	if err != nil {
		return err
	}

	message := append([]byte{unregisterServerMessage}, serverProto...)

	_, err = r.conn.Do("PUBLISH", "channel_servers", message)
	if err != nil {
		return err
	}

	return nil
}

// RegisterRoom register room to redis.
func (r *Redis) RegisterRoom(room *pb.Room) error {
	serverProto, err := proto.Marshal(room)
	if err != nil {
		return err
	}

	message := append([]byte{registerRoomMessage}, serverProto...)

	_, err = r.conn.Do("PUBLISH", "channel_rooms", message)
	if err != nil {
		return err
	}

	return nil
}

// UnregisterRoom unregisters room from redis.
func (r *Redis) UnregisterRoom(room *pb.Room) error {
	serverProto, err := proto.Marshal(room)
	if err != nil {
		return err
	}

	message := append([]byte{unregisterRoomMessage}, serverProto...)

	_, err = r.conn.Do("PUBLISH", "channel_rooms", message)
	if err != nil {
		return err
	}

	return nil
}

// Close is a method to release resources collectively.
func (r *Redis) Close() error {
	return r.conn.Close()
}

// NewRedis is a constructor of Redis. Perform initialization at once.
func NewRedis(hostname string) (*Redis, error) {
	conn, err := redis.Dial("tcp", hostname)
	if err != nil {
		return nil, err
	}
	return &Redis{conn}, nil
}
