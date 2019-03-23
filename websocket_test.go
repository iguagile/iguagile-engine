package main

import (
	"net/http"
	"sync"
	"testing"

	"github.com/iguagile/iguagile-engine/hub"

	"github.com/gorilla/websocket"
)

func TestConnection(t *testing.T) {

	wg := &sync.WaitGroup{}
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

	u := "ws://127.0.0.1:5000/"
	wg.Add(2)

	go func(t *testing.T) {
		ws, _, err := websocket.DefaultDialer.Dial(u, nil)
		if err != nil {
			t.Fatalf("%v", err)
		}

		defer ws.Close()

		_, p, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("%v", err)
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
	}(t)

	go func(t *testing.T) {
		ws, _, err := websocket.DefaultDialer.Dial(u, nil)
		if err != nil {
			t.Errorf("%v", err)
		}
		defer ws.Close()
		data := []byte("11hello1")
		if err := ws.WriteMessage(websocket.BinaryMessage, data); err != nil {
			t.Errorf("%v", err)
		}
		wg.Done()
	}(t)
	wg.Wait()
	srv.Close()
}
