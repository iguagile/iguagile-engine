package iguagile

import (
	"context"
	"crypto/tls"
	"os"
	"sync"
	"testing"

	"github.com/lucas-clemente/quic-go"
)

const quicBenchHost = "127.0.0.1:4101"

func ListenBenchQUIC(b *testing.B) {
	store := NewRedis(os.Getenv("REDIS_HOST"))
	serverID, err := store.GenerateServerID()
	if err != nil {
		b.Fatal(err)
	}
	r := NewRoom(serverID, store)

	tlsConfig, err := generateTLSConfig()
	if err != nil {
		b.Fatal(err)
	}

	listener, err := quic.ListenAddr(quicBenchHost, tlsConfig, nil)
	if err != nil {
		b.Fatal(err)
	}

	go func() {
		for {
			session, err := listener.Accept(context.Background())
			if err != nil {
				b.Error(err)
				continue
			}

			stream, err := session.OpenStreamSync(context.Background())
			if err != nil {
				b.Error(err)
				continue
			}

			r.Serve(stream)
		}
	}()
}

func BenchmarkConnectionQUIC(b *testing.B) {
	ListenBenchQUIC(b)
	wg := &sync.WaitGroup{}
	wg.Add(BenchClients)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"iguagile"},
	}

	for i := 0; i < BenchClients; i++ {
		session, err := quic.DialAddr(quicBenchHost, tlsConfig, nil)
		if err != nil {
			b.Error(err)
		}

		stream, err := session.AcceptStream(context.Background())
		if err != nil {
			b.Error(err)
		}

		client := newBenchClient(stream)
		go client.run(b, wg)
	}

	wg.Wait()
}
