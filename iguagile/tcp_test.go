package iguagile

import (
	"encoding/binary"
	"log"
	"net"
	"os"
	"sync"
	"testing"
)

const host = "127.0.0.1:4000"

func Listen(t *testing.T) {
	store := NewRedis(os.Getenv("REDIS_HOST"))
	serverID, err := store.GenerateServerID()
	if err != nil {
		log.Fatal(err)
	}
	r := NewRoom(serverID, store)

	addr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		t.Errorf("%v", err)
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil && err.Error() != "read: connection reset by peer" {
		t.Errorf("%v", err)
	}
	go func() {
		for {
			conn, err := listener.AcceptTCP()
			if err != nil {
				t.Errorf("%v", err)
			}
			ServeTCP(r, conn)
		}
	}()
}

type testClient struct {
	conn           *net.TCPConn
	isHost         bool
	clientID       uint32
	clientIDByte   []byte
	otherClients   map[uint32]bool
	myObjectID     uint32
	myObjectIDByte []byte
	objects        map[uint32]bool
}

func newTestClient(conn *net.TCPConn) *testClient {
	return &testClient{
		conn:           conn,
		clientID:       0,
		clientIDByte:   make([]byte, 2),
		objects:        make(map[uint32]bool),
		myObjectID:     0,
		myObjectIDByte: make([]byte, 4),
		otherClients:   make(map[uint32]bool),
	}
}

func (c *testClient) run(t *testing.T, waitGroup *sync.WaitGroup) {
	sizeBuf := make([]byte, 2)
	if _, err := c.conn.Read(sizeBuf); err != nil {
		t.Error(err)
	}

	size := int(binary.LittleEndian.Uint16(sizeBuf))
	if size != 3 {
		t.Errorf("invalid length %v", sizeBuf)
	}
	buf := make([]byte, size)
	if _, err := c.conn.Read(buf); err != nil {
		t.Error(err)
	}

	if buf[2] != register {
		t.Errorf("invalid message type %v", buf)
	}

	c.clientID = uint32(binary.LittleEndian.Uint16(buf[:2])) << 16
	c.myObjectID = c.clientID | 1
	binary.LittleEndian.PutUint32(c.myObjectIDByte, c.myObjectID)
	message := append(append([]byte{Server, instantiate}, c.myObjectIDByte...), []byte("iguana")...)
	size = len(message)
	binary.LittleEndian.PutUint16(sizeBuf, uint16(size))
	message = append(sizeBuf, message...)
	if _, err := c.conn.Write(message); err != nil {
		t.Error(err)
	}

	transformMessage := append([]byte{OtherClients, transform}, c.myObjectIDByte...)
	size = len(transformMessage)
	binary.LittleEndian.PutUint16(sizeBuf, uint16(size))
	transformMessage = append(sizeBuf, transformMessage...)
	rpcMessage := append([]byte{OtherClients, rpc}, []byte("iguagile")...)
	size = len(rpcMessage)
	binary.LittleEndian.PutUint16(sizeBuf, uint16(size))
	rpcMessage = append(sizeBuf, rpcMessage...)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		wg.Wait()
		for i := 0; i < 100; i++ {
			if _, err := c.conn.Write(transformMessage); err != nil {
				t.Error(err)
			}
		}

		for i := 0; i < 100; i++ {
			if _, err := c.conn.Write(rpcMessage); err != nil {
				t.Error(err)
			}
		}

		message := append([]byte{Server, destroy}, c.myObjectIDByte...)
		size := len(message)
		sizeBuf := make([]byte, 2)
		binary.LittleEndian.PutUint16(sizeBuf, uint16(size))
		message = append(sizeBuf, message...)
		if _, err := c.conn.Write(message); err != nil {
			t.Error(err)
		}

		waitGroup.Done()
	}()
	for {
		if _, err := c.conn.Read(sizeBuf); err != nil {
			t.Error(err)
		}

		size = int(binary.LittleEndian.Uint16(sizeBuf))
		buf = make([]byte, size)
		if _, err := c.conn.Read(buf); err != nil {
			t.Error(err)
		}

		clientID := uint32(binary.LittleEndian.Uint16(buf)) << 16
		messageType := buf[2]
		switch messageType {
		case newConnection:
			c.otherClients[clientID] = true
		case exitConnection:
			delete(c.otherClients, clientID)
		case instantiate:
			objectID := binary.LittleEndian.Uint32(buf[3:])
			if clientID == c.clientID {
				wg.Done()
				c.myObjectID = objectID
				binary.LittleEndian.PutUint32(c.myObjectIDByte, objectID)
			} else {
				c.objects[objectID] = true
			}
		case destroy:
			objectID := binary.LittleEndian.Uint32(buf[3:])
			if objectID != c.myObjectID {
				delete(c.objects, objectID)
			}
		case migrateHost:
			c.isHost = true
		case transform:
			objectID := binary.LittleEndian.Uint32(buf[3:])
			if !c.objects[objectID] {
				t.Errorf("invalid object id %v", buf)
			}
		case rpc:
			if string(buf[3:]) != "iguagile" {
				t.Errorf("invalid rpc data %v", buf)
			}
		default:
			t.Errorf("invalid message type %v", buf)
		}
	}
}

func TestConnectionTCP(t *testing.T) {
	Listen(t)
	wg := &sync.WaitGroup{}
	const clients = 3
	wg.Add(clients)

	addr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		t.Errorf("%v", err)
	}

	for i := 0; i < clients; i++ {
		conn, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			t.Error(err)
		}
		client := newTestClient(conn)
		go client.run(t, wg)
	}

	wg.Wait()
}
