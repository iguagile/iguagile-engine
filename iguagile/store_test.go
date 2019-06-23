package iguagile

import (
	"testing"
)

var sid int

func TestCanGenerateServerID(t *testing.T) {
	var store = NewRedis("localhost:6379")
	defer func() {
		_ = store.Close()
	}()
	id, err := store.GenerateServerID()
	if err != nil {
		t.Error(err)
	}

	if (id & 0xffff) != 0 {
		t.Errorf("invalid server id %b", id)
	}
	sid = id
}

func TestCanGenerateRoomID(t *testing.T) {
	var store = NewRedis("localhost:6379")
	defer func() {
		_ = store.Close()
	}()

	id, err := store.GenerateRoomID(sid)
	if err != nil {
		t.Error(err)
	}

	if (id & 0xffff0000) != sid {
		t.Errorf("invalid server id %b", id)
	}
}
