package iguagile

import (
	"testing"
)

var sid int

func TestCanGenerateServerID(t *testing.T) {
	var store = NewRedis("localhost:6379")
	id, err := store.GenerateServerID()
	if err != nil {
		t.Error(err)
	}
	if id <= 1>>16 {
		t.Errorf("invalid server id %b", id)
	}
	sid = id
}

func TestCanGenerateRoomID(t *testing.T) {
	var store = NewRedis("localhost:6379")
	id, err := store.GenerateRoomID(sid)
	if err != nil {
		t.Error(err)
	}
	if id <= sid {
		t.Errorf("invalid server id %b", id)
	}
}
