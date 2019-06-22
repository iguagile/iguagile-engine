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

// Redis TODO godoc.
type Redis struct {
	conn redis.Conn
}

// GenerateServerID  TODO godoc.
func (r *Redis) GenerateServerID() (int, error) {
	i, err := redis.Int(r.conn.Do("INCR", "server_id"))
	return i << 16, err
}

// GenerateRoomID  TODO godoc.
func (r *Redis) GenerateRoomID(serverID int) (int, error) {
	i, err := redis.Int(r.conn.Do("INCR", "room_id"))
	return i | serverID, err
}

// Close  TODO godoc.
func (r *Redis) Close() error {
	return r.conn.Close()
}

// NewRedis TODO godoc.
func NewRedis(hostname string) *Redis {
	conn, err := redis.Dial("tcp", hostname)
	if err != nil {
		log.Println(err)
		log.Fatal("failed to connect backend storage.")

	}
	return &Redis{conn}
}
