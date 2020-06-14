package iguagile

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"os"
	"testing"
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
	testData   = []byte("test data")
)

func setupServer() error {
	factory := &RelayServiceFactory{}
	store, err := NewRedis(os.Getenv("REDIS_HOST"))
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

func startServer() error {
	if err := setupServer(); err != nil {
		return err
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	go func() {
		_ = roomServer.Run(listener, grpcPort)
	}()

	return nil
}

func send(writer io.Writer, data []byte) error {
	buf := make([]byte, len(data)+2)
	binary.LittleEndian.PutUint16(buf, uint16(len(data)))
	copy(buf[2:], data)
	_, err := writer.Write(buf)
	return err
}

func receive(reader io.Reader, buf []byte) (int, error) {
	if _, err := reader.Read(buf[:2]); err != nil {
		return 0, err
	}

	size := binary.LittleEndian.Uint16(buf)
	return reader.Read(buf[:int(size)])
}

func verify(writer io.Writer) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf[:], roomID)
	for _, data := range [][]byte{buf, []byte(appName), []byte(appVersion), []byte(password), roomToken} {
		if err := send(writer, data); err != nil {
			return err
		}
	}

	return nil
}

func TestRelayService(t *testing.T) {
	if err := startServer(); err != nil {
		t.Fatal(err)
	}

	_, err := createRoom()
	if err != nil {
		t.Fatal(err)
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}

	if err := verify(conn); err != nil {
		t.Fatal(err)
	}

	if err := send(conn, testData); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, maxMessageSize)
	n, err := receive(conn, buf)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(buf[:n], testData) {
		t.Errorf("invalid data %v, %v", buf[:n], testData)
	}

	t.Logf("%v, %v", buf[:n], testData)
}
