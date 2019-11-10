package iguagile

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"sync"
	"testing"

	"github.com/lucas-clemente/quic-go"
)

const quicTestHost = "127.0.0.1:4001"

type quicListener struct {
	quic.Listener
}

type quicStream struct {
	quic.Stream
}

// LocalAddr is a dummy method to implement listener.
func (s *quicStream) LocalAddr() net.Addr {
	return nil
}

// RemoteAddr is a dummy method to implement listener.
func (s *quicStream) RemoteAddr() net.Addr {
	return nil
}

// Accept accepts quic session and accepts stream.
func (l *quicListener) Accept() (net.Conn, error) {
	session, err := l.Listener.Accept(context.Background())
	if err != nil {
		return nil, err
	}

	s, err := session.OpenStreamSync(context.Background())
	if err != nil {
		return nil, err
	}

	stream := &quicStream{s}
	return stream, nil
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
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{tlsCert},
		NextProtos:         []string{"iguagile"},
	}, nil
}

func jTestConnectionQUIC(t *testing.T) {
	tlsConfig, err := generateTLSConfig()
	if err != nil {
		t.Fatal(err)
	}

	listener, err := quic.ListenAddr(quicTestHost, tlsConfig, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := listen(t, &quicListener{listener}, testClients); err != nil {
		t.Fatal(err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(testClients)

	for i := 0; i < testClients; i++ {
		session, err := quic.DialAddr(quicTestHost, tlsConfig, nil)
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
