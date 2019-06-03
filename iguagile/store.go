package iguagile

import (
	"log"

	"github.com/gomodule/redigo/redis"
)

// Store is an interface for connecting to backend storage and storing data.
type Store interface {
	Send([]byte) error
	Close() error
}

// Redis TODO godoc.
type Redis struct {
	conn   redis.Conn
	roomID []byte
}

// Send  TODO godoc.
func (r *Redis) Send(b []byte) error {
	_, err := r.conn.Do("SET", r.roomID, b)
	return err
}

// Close  TODO godoc.
func (r *Redis) Close() error {
	return r.conn.Close()
}

// NewRedis TODO godoc.
func NewRedis(hostname string, uid []byte) Redis {
	conn, err := redis.Dial("tcp", hostname)
	if err != nil {
		log.Fatal("filed to connect backend storage.")
	}
	return Redis{conn, uid}
}
