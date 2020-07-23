package iguagile

import (
	"os"
	"testing"
)

func TestCanGenerateServerID(t *testing.T) {
	store, err := NewRedis(os.Getenv("REDIS_HOST"))
	if err != nil {
		t.Fatal(err)
	}
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
}
