package iguagile

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"io"
	"log"
	"math/big"
	"os"
	"sync"
	"testing"

	"github.com/lucas-clemente/quic-go"
)

const quicHost = "127.0.0.1:4100"

func ListenQUIC(t *testing.T) {
	store, err := NewRedis(os.Getenv("REDIS_HOST"))
	if err != nil {
		t.Fatal(err)
	}

	rsf := RelayServiceFactory{}
	rs, err := NewRoomServer(rsf, store, quicHost)
	if err != nil {
		t.Fatal(err)
	}
	rc := RoomConfig{
		1,
		"iguana_test",
		"1.0.0",
		"pass",
		100,
		map[string]string{},
		[]byte("aaa"),
	}

	r, err := newRoom(rs, &rc)
	//rss, err := rsf.Create(r)
	if err != nil {
		t.Fatal(err)
	}

	tlsConfig, err := generateTLSConfig()
	if err != nil {
		t.Fatal(err)
	}

	listener, err := quic.ListenAddr(quicHost, tlsConfig, nil)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			session, err := listener.Accept(context.Background())
			if err != nil {
				t.Error(err)
			}

			stream, err := session.OpenStreamSync(context.Background())
			if err != nil {
				t.Error(err)
			}

			err = r.serve(stream)
			if err != nil {
				t.Error(err)
			}
		}
	}()
}

func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"iguagile"},
	}, nil
}

const clients = 3

func TestConnectionQUIC(t *testing.T) {
	ListenQUIC(t)
	wg := &sync.WaitGroup{}
	wg.Add(clients)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"iguagile"},
	}

	for i := 0; i < clients; i++ {
		session, err := quic.DialAddr(quicHost, tlsConfig, nil)
		if err != nil {
			t.Error(err)
		}

		stream, err := session.AcceptStream(context.Background())
		if err != nil {
			t.Error(err)
		}

		client := newTestClient(stream)
		go client.run(t, wg)
	}

	wg.Wait()
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
	message := append(append(c.myObjectIDByte), []byte("iguana")...)
	if err := c.write(message); err != nil {
		t.Error(err)
	}

	// Prepare a transform message and rpc message in advance.
	transformMessage := c.myObjectIDByte
	rpcMessage := []byte("iguagile")

	wg := &sync.WaitGroup{}
	wg.Add(clients)
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

				if err := c.write(objectIDByte); err != nil {
					t.Error(err)
				}
			}
			c.objectsLock.Unlock()

			objectIDByte := make([]byte, 4)
			binary.LittleEndian.PutUint32(objectIDByte, c.myObjectID)

			if err := c.write(objectIDByte); err != nil {
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
		// payload := buf[3:]
		switch messageType {
		case newConnection:
			c.otherClients[clientID] = true
		case exitConnection:
			delete(c.otherClients, clientID)

		case migrateHost:
			c.isHost = true
		default:
			t.Errorf("invalid message type %v", buf)
		}
	}
}
