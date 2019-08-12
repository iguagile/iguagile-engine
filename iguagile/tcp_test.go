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

func ListenTCP(t *testing.T) {
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

type testConn interface {
	read() ([]byte, error)
	write([]byte) error
}

type testTCPConn struct {
	conn *net.TCPConn
}

func (c *testTCPConn) read() ([]byte, error) {
	sizeBuf := make([]byte, 2)
	if _, err := c.conn.Read(sizeBuf); err != nil {
		return nil, err
	}

	size := int(binary.LittleEndian.Uint16(sizeBuf))

	buf := make([]byte, size)
	if _, err := c.conn.Read(buf); err != nil {
		return nil, err
	}

	return buf, nil
}

func (c *testTCPConn) write(message []byte) error {
	size := len(message)
	sizeBuf := make([]byte, 2)
	binary.LittleEndian.PutUint16(sizeBuf, uint16(size))
	data := append(sizeBuf, message...)
	if _, err := c.conn.Write(data); err != nil {
		return err
	}
	return nil
}

type testClientTCP struct {
	conn           testConn
	isHost         bool
	clientID       uint32
	clientIDByte   []byte
	otherClients   map[uint32]bool
	myObjectID     uint32
	myObjectIDByte []byte
	objects        map[uint32]bool
	objectsLock    *sync.Mutex
}

func newTestClient(conn testConn) *testClientTCP {
	return &testClientTCP{
		conn:           conn,
		clientID:       0,
		clientIDByte:   make([]byte, 2),
		objects:        make(map[uint32]bool),
		objectsLock:    &sync.Mutex{},
		myObjectID:     0,
		myObjectIDByte: make([]byte, 4),
		otherClients:   make(map[uint32]bool),
	}
}

const clients = 3

func (c *testClientTCP) run(t *testing.T, waitGroup *sync.WaitGroup) {
	//First receive register message and get client id.
	buf, err := c.conn.read()
	if err != nil {
		t.Error(err)
	}

	if buf[2] != register {
		t.Errorf("invalid message type %v", buf)
	}

	c.clientID = uint32(binary.LittleEndian.Uint16(buf[:2])) << 16

	// Set object id and send instantiate message.
	c.myObjectID = c.clientID | 1
	binary.LittleEndian.PutUint32(c.myObjectIDByte, c.myObjectID)
	message := append(append([]byte{Server, instantiate}, c.myObjectIDByte...), []byte("iguana")...)
	if err := c.conn.write(message); err != nil {
		t.Error(err)
	}

	// Prepare a transform message and rpc message in advance.
	transformMessage := append([]byte{OtherClients, transform}, c.myObjectIDByte...)
	rpcMessage := append([]byte{OtherClients, rpc}, []byte("iguagile")...)

	wg := &sync.WaitGroup{}
	wg.Add(clients)
	go func() {
		// Wait for the object to be instantiated before starting sending messages.
		wg.Wait()
		for i := 0; i < 100; i++ {
			if err := c.conn.write(transformMessage); err != nil {
				t.Error(err)
			}
		}

		for i := 0; i < 100; i++ {
			if err := c.conn.write(rpcMessage); err != nil {
				t.Error(err)
			}
		}

		if c.isHost {
			log.Println("before lock")
			c.objectsLock.Lock()
			log.Println("after lock")
			for objectID := range c.objects {
				objectIDByte := make([]byte, 4)
				binary.LittleEndian.PutUint32(objectIDByte, objectID)
				log.Printf("send request %v\n", objectID)
				message := append([]byte{Server, requestObjectControlAuthority}, objectIDByte...)
				if err := c.conn.write(message); err != nil {
					t.Error(err)
				}
			}
			c.objectsLock.Unlock()

			objectIDByte := make([]byte, 4)
			binary.LittleEndian.PutUint32(objectIDByte, c.myObjectID)
			message := append([]byte{Server, destroy}, objectIDByte...)
			if err := c.conn.write(message); err != nil {
				t.Error(err)
			}
		}
	}()
	for {
		// Start receiving messages.
		buf, err := c.conn.read()
		if err != nil {
			t.Error(err)
		}

		clientID := uint32(binary.LittleEndian.Uint16(buf)) << 16
		messageType := buf[2]
		payload := buf[3:]
		switch messageType {
		case newConnection:
			c.otherClients[clientID] = true
		case exitConnection:
			delete(c.otherClients, clientID)
		case instantiate:
			objectID := binary.LittleEndian.Uint32(payload)
			log.Printf("instantiate %v %v\n", objectID, c.clientID)
			wg.Done()
			if clientID != c.clientID {
				c.objectsLock.Lock()
				c.objects[objectID] = true
				c.objectsLock.Unlock()
			}
		case destroy:
			objectID := binary.LittleEndian.Uint32(payload)
			log.Printf("destroy %v, %v\n", objectID, c.myObjectID)
			if objectID != c.myObjectID {
				c.objectsLock.Lock()
				delete(c.objects, objectID)
				c.objectsLock.Unlock()
			} else {
				waitGroup.Done()
			}
		case migrateHost:
			c.isHost = true
		case requestObjectControlAuthority:
			objectID := binary.LittleEndian.Uint32(payload)
			log.Printf("request %v\n", objectID)
			if objectID != c.myObjectID {
				t.Errorf("invalid object id %v", buf)
				break
			}

			clientIDByte := make([]byte, 4)
			binary.LittleEndian.PutUint32(clientIDByte, clientID)
			message := append(append([]byte{Server, transferObjectControlAuthority}, payload...), clientIDByte...)
			if err := c.conn.write(message); err != nil {
				t.Error(err)
			}
		case transferObjectControlAuthority:
			log.Printf("transfer %v\n", binary.LittleEndian.Uint32(payload))
			message := append([]byte{Server, destroy}, payload...)
			if err := c.conn.write(message); err != nil {
				t.Error(err)
			}
		case transform:
			objectID := binary.LittleEndian.Uint32(payload)
			if objectID == c.myObjectID {
				t.Errorf("invalid object id %v", buf)
			}
		case rpc:
			if string(payload) != "iguagile" {
				t.Errorf("invalid rpc data %v", buf)
			}
		default:
			t.Errorf("invalid message type %v", buf)
		}
	}
}

func TestConnectionTCP(t *testing.T) {
	ListenTCP(t)
	wg := &sync.WaitGroup{}
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
		client := newTestClient(&testTCPConn{conn})
		go client.run(t, wg)
	}

	wg.Wait()
}
