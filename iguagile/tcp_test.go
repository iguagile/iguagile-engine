package iguagile

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"testing"
)

const (
	tcpTestHost = "127.0.0.1:4000"
	maxUser     = 100
	testClients = 3
)

var token = []byte("token")

var config = &RoomConfig{
	RoomID:          0,
	ApplicationName: "test-app",
	Version:         "0.0.0-test",
	Password:        "password",
	MaxUser:         maxUser,
	Token:           token,
}

func listen(tb testing.TB, listener net.Listener, clients int) error {
	store, err := NewRedis(os.Getenv("REDIS_HOST"))
	if err != nil {
		return err
	}

	room, err := NewRoom(store, config)
	if err != nil {
		return err
	}

	go func() {
		for i := 0; i < clients; i++ {
			conn, err := listener.Accept()
			if err != nil {
				tb.Error(err)
			}
			room.Serve(conn)
		}
	}()

	return nil
}

func (c *testClient) read() ([]byte, error) {
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

func (c *testClient) write(message []byte) error {
	size := len(message)
	sizeBuf := make([]byte, 2)
	binary.LittleEndian.PutUint16(sizeBuf, uint16(size))
	data := append(sizeBuf, message...)
	c.writeLock.Lock()
	defer c.writeLock.Unlock()
	if _, err := c.conn.Write(data); err != nil {
		return err
	}
	return nil
}

type testClient struct {
	conn           io.ReadWriteCloser
	isHost         bool
	clientID       uint32
	clientIDByte   []byte
	otherClients   map[uint32]bool
	myObjectID     uint32
	myObjectIDByte []byte
	objects        map[uint32]bool
	objectsLock    *sync.Mutex
	writeLock      *sync.Mutex
}

func newTestClient(conn io.ReadWriteCloser) *testClient {
	return &testClient{
		conn:           conn,
		clientID:       0,
		clientIDByte:   make([]byte, 2),
		objects:        make(map[uint32]bool),
		objectsLock:    &sync.Mutex{},
		myObjectID:     0,
		myObjectIDByte: make([]byte, 4),
		otherClients:   make(map[uint32]bool),
		writeLock:      &sync.Mutex{},
	}
}

func (c *testClient) run(t *testing.T, waitGroup *sync.WaitGroup) {
	//First receive register message and get client id.
	buf, err := c.read()
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
	if err := c.write(message); err != nil {
		t.Error(err)
	}

	// Prepare a transform message and rpc message in advance.
	transformMessage := append([]byte{OtherClients, transform}, c.myObjectIDByte...)
	rpcMessage := append([]byte{OtherClients, rpc}, []byte("iguagile")...)

	wg := &sync.WaitGroup{}
	wg.Add(testClients)
	go func() {
		// Wait for the object to be instantiated before starting sending messages.
		wg.Wait()
		for i := 0; i < 100; i++ {
			if err := c.write(transformMessage); err != nil {
				t.Error(err)
			}
		}

		for i := 0; i < 100; i++ {
			if err := c.write(rpcMessage); err != nil {
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
				if err := c.write(message); err != nil {
					t.Error(err)
				}
			}
			c.objectsLock.Unlock()

			objectIDByte := make([]byte, 4)
			binary.LittleEndian.PutUint32(objectIDByte, c.myObjectID)
			message := append([]byte{Server, destroy}, objectIDByte...)
			if err := c.write(message); err != nil {
				t.Error(err)
			}
		}
	}()
	for {
		// Start receiving messages.
		buf, err := c.read()
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
			if err := c.write(message); err != nil {
				t.Error(err)
			}
		case transferObjectControlAuthority:
			log.Printf("transfer %v\n", binary.LittleEndian.Uint32(payload))
			message := append([]byte{Server, destroy}, payload...)
			if err := c.write(message); err != nil {
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
	listener, err := net.Listen("tcp", tcpTestHost)
	if err != nil {
		t.Fatal(err)
	}

	if err := listen(t, listener, testClients); err != nil {
		t.Fatal(err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(testClients)

	for i := 0; i < testClients; i++ {
		conn, err := net.Dial("tcp", tcpTestHost)
		if err != nil {
			t.Error(err)
		}
		client := newTestClient(conn)
		go client.run(t, wg)
	}

	wg.Wait()
}
