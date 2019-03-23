package main

import (
	"net/http"
	"reflect"
	"sync"
	"testing"

	"github.com/iguagile/iguagile-engine/hub"

	"github.com/gorilla/websocket"
)

func TestConnection(t *testing.T) {

	wg := &sync.WaitGroup{}
	wg.Add(1)
	srv := &http.Server{
		Addr: "127.0.0.1:5000",
	}

	wg.Add(1)
	go func() {

		h := hub.NewHub()
		go h.Run()
		f := func(w http.ResponseWriter, r *http.Request) {
			hub.ServeWs(h, w, r)
		}
		srv.Handler = http.HandlerFunc(f)

		if err := srv.ListenAndServe(); err != nil {
			wg.Done()
			t.Fatal(err)
		}
		srv.RegisterOnShutdown(func() {
			wg.Done()
		})
	}()

	u := "ws://127.0.0.1:5000/"

	data := []byte("hello1")
	go func() {
		ws, _, err := websocket.DefaultDialer.Dial(u, nil)
		if err != nil {
			t.Fatalf("%v", err)
		}

		defer ws.Close()

		_, p, err := ws.ReadMessage()
		if err != nil {
			wg.Done()

			t.Fatalf("%v", err)
		}
		if !reflect.DeepEqual(data, p) {
			wg.Done()
			srv.Close()
			println(string(p))
			t.Fatalf("bad message")
		}

		wg.Done()
		srv.Close()
	}()

	wg.Add(1)
	go func() {
		ws, _, err := websocket.DefaultDialer.Dial(u, nil)
		if err != nil {
			wg.Done()
			t.Fatalf("%v", err)
		}
		defer ws.Close()

		if err := ws.WriteMessage(websocket.BinaryMessage, data); err != nil {
			wg.Done()
			t.Fatalf("%v", err)
		}
		wg.Done()
	}()
	wg.Wait()

}
