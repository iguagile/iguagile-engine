package iguagile

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"os"
	"testing"
)

const (
	address = "localhost:8080"
	apiAddr = "localhost:8081"

	serverID   = 1 << 16
	roomID     = 1 | serverID
	appName    = "iguana online"
	appVersion = "0.0.0 beta"
	password   = "******"
)

var (
	engine    *Engine
	roomToken = []byte{1}
	testData  = []byte("test data")
)

func setupServer() error {
	factory := &RelayServiceFactory{}
	store, err := NewRedis(os.Getenv("REDIS_HOST"))
	if err != nil {
		return err
	}

	engine = New(factory, store)

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

	room, err := newRoom(engine, conf)
	if err != nil {
		return nil, err
	}

	service, err := engine.factory.Create(room)
	if err != nil {
		return nil, err
	}
	room.service = service
	engine.rooms.Store(roomID, room)

	return room, nil
}

func startServer() error {
	if err := setupServer(); err != nil {
		return err
	}

	tlsConf, err := generateTLSConfig()
	if err != nil {
		return err
	}

	go func() {
		_ = engine.Start(context.Background(), address, apiAddr, tlsConf)
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
		Certificates:       []tls.Certificate{tlsCert},
		NextProtos:         []string{"iguagile-test"},
		InsecureSkipVerify: true,
	}, nil
}
