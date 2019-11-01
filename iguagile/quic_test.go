package iguagile

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"sync"
	"testing"

	"github.com/lucas-clemente/quic-go"
)

const quicHost = "127.0.0.1:4100"

func ListenQUIC(t *testing.T) {
	store := NewRedis(os.Getenv("REDIS_HOST"))
	serverID, err := store.GenerateServerID()
	if err != nil {
		t.Fatal(err)
	}
	r := NewRoom(serverID, store)

	tlsConfig, err := generateTLSConfig()
	if err != nil {
		t.Fatal(err)
	}

	listener, err := quic.ListenAddr(quicHost, tlsConfig, nil)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			session, err := listener.Accept(context.Background())
			if err != nil {
				t.Error(err)
				continue
			}

			stream, err := session.OpenStreamSync(context.Background())
			if err != nil {
				t.Error(err)
				continue
			}

			r.Serve(stream)
		}
	}()
}

func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"iguagile"},
	}, nil
}

func TestConnectionQUIC(t *testing.T) {
	ListenQUIC(t)
	wg := &sync.WaitGroup{}
	wg.Add(clients)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"iguagile"},
	}

	for i := 0; i < clients; i++ {
		session, err := quic.DialAddr(quicHost, tlsConfig, nil)
		if err != nil {
			t.Error(err)
		}

		stream, err := session.AcceptStream(context.Background())
		if err != nil {
			t.Error(err)
		}

		client := newTestClient(stream)
		go client.run(t, wg)
	}

	wg.Wait()
}
