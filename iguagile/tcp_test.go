package iguagile

import (
	"log"
	"net"
	"os"
	"reflect"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/iguagile/iguagile-engine/data"
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

const (
	RPC       = 2
	Transform = 3
)

func TestConnectionTCP(t *testing.T) {
	testData := []struct {
		send []byte
		want []byte
	}{
		{append([]byte{OtherClients, RPC}, "iguana"...), []byte("iguana")},
		{append([]byte{OtherClients, Transform}, "agile"...), []byte("agile")},
	}

	Listen(t)

	addr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		t.Errorf("%v", err)
	}

	rec, err := net.DialTCP("tcp", nil, addr)
	if err != nil && err.Error() != "use of closed network connection" {
		t.Errorf("%v", err)
	}
	defer func() {
		err := rec.Close()
		if err != nil {
			t.Log(err)
		}
	}()

	send, err := net.DialTCP("tcp", nil, addr)
	if err != nil && err.Error() != "use of closed network connection" {
		t.Errorf("%v", err)
	}
	defer func() {
		err := send.Close()
		if err != nil {
			t.Log(err)
		}
	}()

	for i := 0; i < 10; i++ {
		for _, v := range testData {
			wg := &sync.WaitGroup{}
			wg.Add(2)
			go receiverTCP(rec, t, wg, v.want)
			go senderTCP(send, t, wg, v.send)
			wg.Wait()
		}

	}
}

func receiverTCP(conn *net.TCPConn, t *testing.T, wg *sync.WaitGroup, want []byte) {
OUTER:
	for {
		sizeBuf := make([]byte, 2)
		_, err := conn.Read(sizeBuf)
		if err != nil {
			t.Errorf("%v", err)
		}

		size := int(sizeBuf[0]) + int(sizeBuf[1])<<8
		buf := make([]byte, size)
		n, err := conn.Read(buf)
		if err != nil {
			t.Errorf("%v", err)
		}
		if n != size {
			t.Errorf("data size does not match")
		}

		bin, err := data.NewBinaryData(buf, data.Outbound)
		if err != nil {
			t.Error(err)
		}

		switch bin.MessageType {
		case data.NewConnect:
			id, err := uuid.FromBytes(bin.UUID)
			if err != nil {
				t.Error(err)
			}
			t.Logf("new client %s", id)
			continue OUTER
		case data.ExitConnect:
			id, err := uuid.FromBytes(bin.UUID)
			if err != nil {
				t.Error(err)
			}
			t.Logf("client exit %s", id)
			continue OUTER
		default:
			t.Logf("%s\n", bin.Payload)
			if !reflect.DeepEqual(want, bin.Payload) {
				t.Error("miss match message")
				t.Errorf("%v\n", bin.Payload)
				t.Errorf("%s\n", bin.Payload)
			}
			t.Log(string(bin.Payload))

			wg.Done()
			break OUTER
		}
	}
}

func senderTCP(conn *net.TCPConn, t *testing.T, wg *sync.WaitGroup, send []byte) {
	size := len(send)
	buf := append([]byte{byte(size & 255), byte(size >> 8)}, send...)
	_, err := conn.Write(buf)
	if err != nil {
		t.Errorf("%v", err)
	}
	wg.Done()
}
