package iguagile

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var uri = url.URL{Scheme: "ws", Host: "127.0.0.1:5000", Path: "/"}

func ListenWebsocket(t *testing.T) {
	srv := &http.Server{
		Addr: uri.Host,
	}

	store := NewRedis(os.Getenv("REDIS_HOST"))
	serverID, err := store.GenerateServerID()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = store.Close()
	}()
	room := NewRoom(serverID, store)
	f := func(writer http.ResponseWriter, request *http.Request) {
		ServeWebsocket(room, writer, request)
	}
	srv.Handler = http.HandlerFunc(f)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			t.Errorf("%v", err)
		}
		_ = srv.Close()
	}()
}

type testWebsocketConn struct {
	conn *websocket.Conn
}

func (c *testWebsocketConn) read() ([]byte, error) {
	log.Printf("before read")
	_, message, err := c.conn.ReadMessage()
	log.Printf("after read")
	return message, err
}

func (c *testWebsocketConn) write(message []byte) error {
	log.Printf("write")
	return c.conn.WriteMessage(websocket.BinaryMessage, message)
}

func TestConnectionWebsocket(t *testing.T) {
	ListenWebsocket(t)
	time.Sleep(time.Second)

	log.Printf("listen websocket")
	wg := &sync.WaitGroup{}
	wg.Add(clients)

	for i := 0; i < clients; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(uri.String(), nil)
		if err != nil {
			t.Error(err)
		}
		client := newTestClient(&testWebsocketConn{conn})
		log.Printf("run")
		go client.run(t, wg)
	}

	wg.Wait()
}
