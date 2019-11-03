package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log"
	"math/big"

	"github.com/iguagile/iguagile-engine/iguagile"
	"github.com/lucas-clemente/quic-go"
)

func main() {
	store, err := iguagile.NewDummyStore()
	if err != nil {
		log.Fatal(err)
	}
	serverID, err := store.GenerateServerID()
	if err != nil {
		log.Fatal(err)
	}
	r := iguagile.NewRoom(serverID, store)

	tlsConfig, err := generateTLSConfig()
	if err != nil {
		log.Fatal(err)
	}

	listener, err := quic.ListenAddr("localhost:5100", tlsConfig, nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("ListenQUIC")
	for {
		session, err := listener.Accept(context.Background())
		if err != nil {
			log.Println(err)
			continue
		}

		stream, err := session.OpenStreamSync(context.Background())
		if err != nil {
			log.Println(err)
			continue
		}

		r.Serve(stream)
	}
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
