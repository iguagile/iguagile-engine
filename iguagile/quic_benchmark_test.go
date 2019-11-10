package iguagile

import (
	"context"
	"sync"
	"testing"

	"github.com/lucas-clemente/quic-go"
)

const quicBenchHost = "127.0.0.1:4101"

func BenchmarkConnectionQUIC(b *testing.B) {
	tlsConfig, err := generateTLSConfig()
	if err != nil {
		b.Fatal(err)
	}

	listener, err := quic.ListenAddr(quicBenchHost, tlsConfig, nil)
	if err != nil {
		b.Fatal(err)
	}

	if err := listen(b, &quicListener{listener}); err != nil {
		b.Fatal(err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(BenchClients)

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
