package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/iguagile/iguagile-engine/hub"

	"github.com/gorilla/websocket"
)

func TestConnection(t *testing.T) {

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
			t.Error(err)
		}
	}(t)

	time.Sleep(1 * time.Second)
	fmt.Println("RUN")
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func(t *testing.T, wg *sync.WaitGroup) {
		u := url.URL{Scheme: "ws", Host: "127.0.0.1:5000", Path: "/ws"}
		ws, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
		//ws, resp, err := dialer.Dial(u, nil)
		if err == websocket.ErrBadHandshake {
			t.Logf("handshake failed with status %d", resp.StatusCode)
			t.Fatalf("%v", err)
		}

		defer func(ws *websocket.Conn, wg *sync.WaitGroup) {
			if err := ws.Close(); err != nil {
				t.Error(err)
			}
		}(ws, wg)

		messageType, p, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("%v", err)
		}

		if messageType != websocket.BinaryMessage {
			t.Fail()
		}
		// remove uuid
		received := p[16:]
		// remove mesType
		data := received[1:]

		if "hello1" != string(data) {
			t.Error("bad message")
			t.Errorf("%v", data)
			t.Errorf("%s", data)
		}
		t.Log(string(data))
		wg.Done()

	}(t, wg)

	go func(t *testing.T, wg *sync.WaitGroup) {
		u := url.URL{Scheme: "ws", Host: "127.0.0.1:5000", Path: "/"}
		ws, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err == websocket.ErrBadHandshake {
			t.Logf("handshake failed with status %d", resp.StatusCode)
			t.Errorf("%v", err)
		}
		defer func(ws *websocket.Conn, wg *sync.WaitGroup) {
			if err := ws.Close(); err != nil {
				t.Error(err)
			}
		}(ws, wg)

		data := []byte("11hello1")
		if err := ws.WriteMessage(websocket.BinaryMessage, data); err != nil {
			t.Errorf("%v", err)
		}
		wg.Done()

	}(t, wg)
	wg.Wait()
	srv.Close()
}
