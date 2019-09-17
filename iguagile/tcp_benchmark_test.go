package iguagile

import (
	"encoding/binary"
	"log"
	"net"
	"os"
	"sync"
	"testing"
)

func ListenBenchTCP(b *testing.B) {
	store := NewRedis(os.Getenv("REDIS_HOST"))
	serverID, err := store.GenerateServerID()
	if err != nil {
		log.Fatal(err)
	}
	r := NewRoom(serverID, store)

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:4021")
	if err != nil {
		b.Errorf("%v", err)
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil && err.Error() != "read: connection reset by peer" {
		b.Errorf("%v", err)
	}
	go func() {
		for {
			conn, err := listener.AcceptTCP()
			if err != nil {
				b.Errorf("%v", err)
			}
			ServeTCP(r, conn)
		}
	}()
}

type benchConn interface {
	read() ([]byte, error)
	write([]byte) error
}

type benchTCPConn struct {
	conn *net.TCPConn
	*sync.Mutex
}

func (c *benchTCPConn) read() ([]byte, error) {
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

func (c *benchTCPConn) write(message []byte) error {
	size := len(message)
	sizeBuf := make([]byte, 2)
	binary.LittleEndian.PutUint16(sizeBuf, uint16(size))
	data := append(sizeBuf, message...)
	c.Lock()
	defer c.Unlock()
	if _, err := c.conn.Write(data); err != nil {
		return err
	}
	return nil
}

type benchClient struct {
	conn           benchConn
	isHost         bool
	clientID       uint32
	clientIDByte   []byte
	otherClients   map[uint32]bool
	myObjectID     uint32
	myObjectIDByte []byte
	objects        map[uint32]bool
	objectsLock    *sync.Mutex
}

func newBenchClient(conn benchConn) *benchClient {
	return &benchClient{
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

const BenchClients = 1

func (c *benchClient) run(b *testing.B, waitGroup *sync.WaitGroup) {
	//First receive register message and get client id.
	buf, err := c.conn.read()
	if err != nil {
		b.Error(err)
	}

	if buf[2] != register {
		b.Errorf("invalid message type %v", buf)
	}

	c.clientID = uint32(binary.LittleEndian.Uint16(buf[:2])) << 16

	// Set object id and send instantiate message.
	c.myObjectID = c.clientID | 1
	binary.LittleEndian.PutUint32(c.myObjectIDByte, c.myObjectID)
	message := append(append([]byte{Server, instantiate}, c.myObjectIDByte...), []byte("iguana")...)
	if err := c.conn.write(message); err != nil {
		b.Error(err)
	}

	// Prepare a transform message and rpc message in advance.
	transformMessage := append([]byte{OtherClients, transform}, c.myObjectIDByte...)
	rpcMessage := append([]byte{OtherClients, rpc}, []byte("iguagile")...)

	wg := &sync.WaitGroup{}
	wg.Add(BenchClients)
	go func() {
		// Wait for the object to be instantiated before starting sending messages.
		wg.Wait()
		for i := 0; i < 100; i++ {
			if err := c.conn.write(transformMessage); err != nil {
				b.Error(err)
			}
		}

		for i := 0; i < 100; i++ {
			if err := c.conn.write(rpcMessage); err != nil {
				b.Error(err)
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
					b.Error(err)
				}
			}
			c.objectsLock.Unlock()

			objectIDByte := make([]byte, 4)
			binary.LittleEndian.PutUint32(objectIDByte, c.myObjectID)
			message := append([]byte{Server, destroy}, objectIDByte...)
			if err := c.conn.write(message); err != nil {
				b.Error(err)
			}
		}
	}()
	for {
		// Start receiving messages.
		buf, err := c.conn.read()
		if err != nil {
			b.Error(err)
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
				b.Errorf("invalid object id %v", buf)
				break
			}

			clientIDByte := make([]byte, 4)
			binary.LittleEndian.PutUint32(clientIDByte, clientID)
			message := append(append([]byte{Server, transferObjectControlAuthority}, payload...), clientIDByte...)
			if err := c.conn.write(message); err != nil {
				b.Error(err)
			}
		case transferObjectControlAuthority:
			log.Printf("transfer %v\n", binary.LittleEndian.Uint32(payload))
			message := append([]byte{Server, destroy}, payload...)
			if err := c.conn.write(message); err != nil {
				b.Error(err)
			}
		case transform:
			objectID := binary.LittleEndian.Uint32(payload)
			if objectID == c.myObjectID {
				b.Errorf("invalid object id %v", buf)
			}
		case rpc:
			if string(payload) != "iguagile" {
				b.Errorf("invalid rpc data %v", buf)
			}
		default:
			b.Errorf("invalid message type %v", buf)
		}
	}
}

func BenchmarkConnectionTCP(b *testing.B) {
	os.Setenv("REDIS_HOST", "localhost:6379")
	ListenBenchTCP(b)
	wg := &sync.WaitGroup{}
	wg.Add(BenchClients)

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:4021")
	if err != nil {
		b.Errorf("%v", err)
	}

	for i := 0; i < BenchClients; i++ {
		conn, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			b.Error(err)
		}
		client := newBenchClient(&benchTCPConn{conn, &sync.Mutex{}})
		go client.run(b, wg)
	}

	wg.Wait()
}
