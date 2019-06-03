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

// TODO godoc.
type Redis struct {
	conn   redis.Conn
	roomID []byte
}

func (r *Redis) Send(b []byte) error {
	_, err := r.conn.Do("SET", r.roomID, b)
	return err
}

func (r *Redis) Close() error {
	return r.conn.Close()
}

// TODO godoc.
func NewRedis(hostname string, port int, uid []byte) Redis {
	conn, err := redis.Dial("tcp", "localhost:6379")
	if err != nil {
		log.Fatal("filed to connect backend storage.")
	}
	return Redis{conn, uid}
}
