package iguagile

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"testing"

	"github.com/lucas-clemente/quic-go"
)

const (
	address  = "localhost:12345"
	grpcPort = 11111

	serverID   = 1 << 16
	roomID     = 1 | serverID
	appName    = "iguana online"
	appVersion = "0.0.0 beta"
	password   = "******"
)

var (
	roomServer *RoomServer
	roomToken  = []byte{1}
)

func setupServer() error {
	factory := &RelayServiceFactory{}
	store, err := NewRedis(":6379")
	if err != nil {
		return err
	}

	roomServer, err = NewRoomServer(factory, store, address)
	if err != nil {
		return err
	}

	return nil
}

func createRoom() (*Room, error) {
	conf := &RoomConfig{
		RoomID:          roomID,
		ApplicationName: appName,
		Version:         appVersion,
		Password:        password,
		MaxUser:         10,
		Token:           roomToken,
	}

	room, err := newRoom(roomServer, conf)
	if err != nil {
		return nil, err
	}

	service, err := roomServer.factory.Create(room)
	if err != nil {
		return nil, err
	}
	room.service = service
	roomServer.rooms.Store(roomID, room)

	return room, nil
}

func ListenQUIC(t *testing.T) {
	if err := setupServer(); err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_ = roomServer.Run(listener, grpcPort)
	}()

}

const clients = 1

func TestConnectionQUIC(t *testing.T) {
	ListenQUIC(t)
	wg := &sync.WaitGroup{}
	wg.Add(clients)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"iguagile"},
	}

	room, err := createRoom()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < clients; i++ {
		session, err := quic.DialAddr(address, tlsConfig, nil)
		if err != nil {
			t.Error(err)
		}

		stream, err := session.AcceptStream(context.Background())
		if err != nil {
			t.Error(err)
		}

		client, err := NewClient(room, stream)
		if err != nil {
			t.Error(err)
		}
		err = room.register(client)
		if err != nil {
			t.Error(err)
		}
		go run(client, room, t, wg)
	}

	wg.Wait()
}

func run(c *Client, room *Room, t *testing.T, wg *sync.WaitGroup) {
	room.SendToAllClients(c.id, []byte("a"))
	// room.unregister(c)
	// c.write([]byte("aaa"))
	// c.Close()

	wg.Done()

}
