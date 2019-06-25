package iguagile

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/iguagile/iguagile-engine/data"
)

var uri = url.URL{Scheme: "ws", Host: "127.0.0.1:5000", Path: "/"}

func NewServer(t *testing.T) *http.Server {
	srv := &http.Server{
		Addr: uri.Host,
	}

	go func(t *testing.T) {
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

		if err := srv.ListenAndServe(); err != nil {
			t.Errorf("%v", err)
		}
	}(t)

	return srv
}

func TestConnectionWebsocket(t *testing.T) {
	testData := []struct {
		send []byte
		want []byte
	}{
		{append([]byte{OtherClients, RPC}, "iguana"...), []byte("iguana")},
		{append([]byte{OtherClients, Transform}, "agile"...), []byte("agile")},
		{append([]byte{OtherClients, Transform}, "iguagile"...), []byte("iguagile")},
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
		err := wsRec.Close()
		if err != nil {
			t.Log(err)
		}
	}()

	wsSend, resp, err := websocket.DefaultDialer.Dial(uri.String(), nil)
	if err != nil {
		t.Errorf("%v", err)
		t.Errorf("%v", resp)
	}
	defer func() {
		err := wsSend.Close()
		if err != nil {
			t.Log(err)
		}
	}()

	// THIS IS TEST CORE.
	for i := 0; i < 10; i++ {
		for _, v := range testData {
			wg := &sync.WaitGroup{}
			wg.Add(2)
			go receiverWebsocket(wsRec, t, wg, v.want)
			go senderWebsocket(wsSend, t, wg, v.send)
			wg.Wait()
		}
	}
	// wait senderWebsocket and receiverWebsocket done
	err = wsSend.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil && err.Error() != "websocket: close 1000 (normal)" {
		t.Errorf("%v", err)
	}

	err = wsRec.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	if err != nil && err.Error() != "websocket: close 1000 (normal)" {
		t.Errorf("%v", err)
	}

}

func receiverWebsocket(ws *websocket.Conn, t *testing.T, wg *sync.WaitGroup, want []byte) {

OUTER:
	for {

		messageType, p, err := ws.ReadMessage()
		if err != nil {
			t.Errorf("%v", err)
		}

		if messageType != websocket.BinaryMessage {
			t.Error("support binary message only")
		}

		bin, err := data.NewBinaryData(p, data.Outbound)
		if err != nil {
			t.Error(err)
		}

		switch bin.MessageType {
		case data.NewConnect:
			id := bin.ID[0] | (bin.ID[1] << 8)
			t.Logf("new client %x", id)
			continue OUTER
		case data.ExitConnect:
			id := bin.ID[0] | (bin.ID[1] << 8)
			t.Logf("client exit %x", id)
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

func senderWebsocket(ws *websocket.Conn, t *testing.T, wg *sync.WaitGroup, send []byte) {
	if err := ws.WriteMessage(websocket.BinaryMessage, send); err != nil {
		t.Errorf("%v", err)
	}

	wg.Done()
}
