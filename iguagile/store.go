package iguagile

import (
	"log"

	"github.com/gomodule/redigo/redis"
)

// Store is an interface for connecting to backend storage and storing data.
type Store interface {
	Close() error
	GenerateRoomID(serverID int) (int, error)
	GenerateServerID() (int, error)
}

// Redis is a structure that wraps goredis to make it easy to use.
type Redis struct {
	conn redis.Conn
}

// GenerateServerID is a method to number unique ServerID.
func (r *Redis) GenerateServerID() (int, error) {
	// TODO CHECK TO 1 << 16 over.
	i, err := redis.Int(r.conn.Do("INCR", "server_id"))
	return i << 16, err
}

// GenerateRoomID numbers unique RoomIDs.
func (r *Redis) GenerateRoomID(serverID int) (int, error) {
	i, err := redis.Int(r.conn.Do("INCR", "room_id"))
	return i | serverID, err
}

// Close is a method to release resources collectively.
func (r *Redis) Close() error {
	return r.conn.Close()
}

// NewRedis is a constructor of Redis. Perform initialization at once.
func NewRedis(hostname string) *Redis {
	conn, err := redis.Dial("tcp", hostname)
	if err != nil {
		log.Println(err)
		log.Fatal("failed to connect backend storage.")

	}
	return &Redis{conn}
}
