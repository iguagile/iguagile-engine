package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/iguagile/iguagile-engine/hub"
)

const binaryUUIDLength = 16
const messageTypeLength = 1

func NewServer(t *testing.T) *http.Server {
	srv := &http.Server{
		Addr: "127.0.0.1:5000",
	}

	go func(t *testing.T) {

		h := hub.NewHub()
		go h.Run()
		f := func(w http.ResponseWriter, r *http.Request) {
			hub.ServeWs(h, w, r)
		}
		srv.Handler = http.HandlerFunc(f)

		if err := srv.ListenAndServe(); err != nil {
			t.Log(err)
		}
	}(t)

	return srv
}

func TestConnection(t *testing.T) {
	testData := []struct {
		send string
		want string
	}{
		{"00hello1", "hello1"},
		{"00MSG", "MSG"},
		{"00HOGE", "HOGE"},
	}

	srv := NewServer(t)

	time.Sleep(200 * time.Millisecond)
	fmt.Println("RUN")

	for i := 0; i < 3; i++ {

		for _, v := range testData {
			wg := &sync.WaitGroup{}
			wg.Add(2)
			go receiver(t, wg, v.want)
			go sender(t, wg, v.send)
			wg.Wait()
		}

	}
	srv.Close()
}

func receiver(t *testing.T, wg *sync.WaitGroup, want string) {
	u := url.URL{Scheme: "ws", Host: "127.0.0.1:5000", Path: "/"}
	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

	if err != nil {
		t.Fatalf("%v", err)
	}

	messageType, p, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("%v", err)
	}

	if messageType != websocket.BinaryMessage {
		t.Error("support binary message only")
	}
	t.Log(len(p))
	t.Logf("%v\n", p)
	t.Logf("%s\n", p)
	// remove uuid
	received := p[binaryUUIDLength:]
	// remove mesType
	data := received[messageTypeLength:]

	if want != string(data) {
		t.Error("bad message")
		t.Errorf("%v\n", data)
		t.Errorf("%s\n", data)
	}
	t.Log(string(data))

	// receiver done
	wg.Done()

	// wait sender and receivers done
	wg.Wait()
	if err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		t.Errorf("%v", err)
	}
	time.Sleep(50 * time.Microsecond)
	_ = ws.Close()
}

func sender(t *testing.T, wg *sync.WaitGroup, send string) {

	u := url.URL{Scheme: "ws", Host: "127.0.0.1:5000", Path: "/"}
	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Errorf("%v", err)
	}

	data := []byte(send)
	if err := ws.WriteMessage(websocket.BinaryMessage, data); err != nil {
		t.Errorf("%v", err)
	}

	// sender done
	wg.Done()

	// wait sender and receiver done
	wg.Wait()
	if err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		t.Errorf("%v", err)
	}

	time.Sleep(50 * time.Millisecond)
	_ = ws.Close()

}
