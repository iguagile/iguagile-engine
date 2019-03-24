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

func TestConnection(t *testing.T) {

	const binaryUUIDLength = 16
	const messageTypeLength = 1

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

	time.Sleep(200 * time.Millisecond)
	fmt.Println("RUN")

	for i := 0; i < 20; i++ {
		wg := &sync.WaitGroup{}
		wg.Add(2)

		go func(t *testing.T, wg *sync.WaitGroup) {
			u := url.URL{Scheme: "ws", Host: "127.0.0.1:5000", Path: "/"}
			ws, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)

			if err != nil {
				t.Logf("handshake failed with status %d", resp.StatusCode)
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

			if "hello1" != string(data) {
				t.Error("bad message")
				t.Errorf("%v\n", data)
				t.Errorf("%s\n", data)
			}
			t.Log(string(data))

			wg.Done()
			wg.Wait()
			if err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
				t.Errorf("%v", err)
			}
			time.Sleep(50 * time.Microsecond)
			_ = ws.Close()

		}(t, wg)

		go func(t *testing.T, wg *sync.WaitGroup) {
			u := url.URL{Scheme: "ws", Host: "127.0.0.1:5000", Path: "/"}
			ws, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
			if err != nil {
				t.Logf("handshake failed with status %d", resp.StatusCode)
				t.Errorf("%v", err)
			}

			data := []byte("00hello1")
			if err := ws.WriteMessage(websocket.BinaryMessage, data); err != nil {
				t.Errorf("%v", err)
			}
			t.Logf("send %v %s\n", data, data)

			wg.Done()
			wg.Wait()
			if err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
				t.Errorf("%v", err)
			}

			time.Sleep(50 * time.Millisecond)
			_ = ws.Close()
		}(t, wg)
		wg.Wait()
		time.Sleep(200 * time.Millisecond)

	}
	srv.Close()
}
