package main

import (
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

var uri = url.URL{Scheme: "ws", Host: "127.0.0.1:5000", Path: "/"}

func NewServer(t *testing.T) *http.Server {
	srv := &http.Server{
		Addr: uri.Host,
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
	defer func() {
		if err := srv.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	time.Sleep(200 * time.Millisecond)

	wsRec, resp, err := websocket.DefaultDialer.Dial(uri.String(), nil)
	if err != nil {
		t.Errorf("%v", err)
		t.Errorf("%v", resp)
	}
	defer func() {
		_ = wsRec.Close()
	}()
	wsSend, resp, err := websocket.DefaultDialer.Dial(uri.String(), nil)
	if err != nil {
		t.Errorf("%v", err)
		t.Errorf("%v", resp)
	}
	defer func() {
		_ = wsSend.Close()
	}()

	// THIS IS TEST CORE.
	for i := 0; i < 10; i++ {
		for _, v := range testData {
			wg := &sync.WaitGroup{}
			wg.Add(2)
			go receiver(wsRec, t, wg, v.want)
			go sender(wsSend, t, wg, v.send)
			wg.Wait()
		}

	}
	// wait sender and receiver done
	if err := wsSend.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		t.Errorf("%v", err)
	}
	if err := wsRec.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		t.Errorf("%v", err)
	}
	time.Sleep(50 * time.Microsecond)

}

func receiver(ws *websocket.Conn, t *testing.T, wg *sync.WaitGroup, want string) {

	messageType, p, err := ws.ReadMessage()
	if err != nil {
		t.Errorf("%v", err)
	}

	if messageType != websocket.BinaryMessage {
		t.Error("support binary message only")
	}

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

}

func sender(ws *websocket.Conn, t *testing.T, wg *sync.WaitGroup, send string) {

	ws, _, err := websocket.DefaultDialer.Dial(uri.String(), nil)
	if err != nil {
		t.Errorf("%v", err)
	}

	data := []byte(send)
	if err := ws.WriteMessage(websocket.BinaryMessage, data); err != nil {
		t.Errorf("%v", err)
	}

	// sender done
	wg.Done()

}
