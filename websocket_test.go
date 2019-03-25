package main

import (
	"log"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/iguagile/iguagile-engine/data"

	"github.com/google/uuid"

	"github.com/gorilla/websocket"
	"github.com/iguagile/iguagile-engine/hub"
)

// Message types
const (
	systemMessage = iota
	dataMessage
)

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
		send []byte
		want []byte
	}{
		{append([]byte{1, 1, 0}, []byte("hello")...), []byte("hello")},
		{append([]byte{1, 1, 0}, []byte("MSG")...), []byte("MSG")},
		{append([]byte{1, 1, 0}, []byte("HOGE")...), []byte("HOGE")},
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
	time.Sleep(50 * time.Microsecond)
	if err := wsRec.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		t.Errorf("%v", err)
	}

}

func receiver(ws *websocket.Conn, t *testing.T, wg *sync.WaitGroup, want []byte) {

OUTER:
	for {

		messageType, p, err := ws.ReadMessage()
		if err != nil {
			t.Errorf("%v", err)
		}

		if messageType != websocket.BinaryMessage {
			t.Error("support binary message only")
		}

		bin, err := data.NewBinaryData(p)
		if err != nil {
			t.Error(err)

		}

		// SKIP SYSTEM MESSAGE
		switch bin.MessageType {
		case systemMessage:
			// perse subtype
			id, err := uuid.FromBytes(bin.UUID)
			if err != nil {
				log.Fatal(err)
			}
			switch bin.SubType {
			case 0:
				t.Logf("new client %s", id)
			default:
				t.Logf("client exit %s", id)
			}

			continue OUTER
		case dataMessage:
			t.Logf("%s\n", bin.Payload)
			if !reflect.DeepEqual(want, bin.Payload) {
				t.Error("miss match message")
				t.Errorf("%v\n", bin.Payload)
				t.Errorf("%s\n", bin.Payload)
			}
			t.Log(string(bin.Payload))

			// ws done
			wg.Done()
			break OUTER
		}
	}
}

func sender(ws *websocket.Conn, t *testing.T, wg *sync.WaitGroup, send []byte) {
	if err := ws.WriteMessage(websocket.BinaryMessage, send); err != nil {
		t.Errorf("%v", err)
	}

	// ws done
	wg.Done()

}
