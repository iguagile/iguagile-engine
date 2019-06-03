package iguagile

import (
	"github.com/gomodule/redigo/redis"
)

type Store interface {
	Send([]byte) error
	Close() error
}

type Redis struct {
	conn redis.Conn
	id   []byte
}

func (r *Redis) Send(b []byte) error {
	_, err := r.conn.Do("SET", r.id, b)
	return err
}

func (r *Redis) Close() error {
	return r.conn.Close()
}
