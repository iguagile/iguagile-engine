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
	"os"

	"github.com/iguagile/iguagile-engine/iguagile"
)

func main() {
	factory := &iguagile.RelayServiceFactory{}
	address := os.Getenv("ROOM_HOST")
	apiAddr := os.Getenv("GRPC_HOST")

	store, err := iguagile.NewRedis(os.Getenv("REDIS_HOST"))
	if err != nil {
		log.Fatal(err)
	}

	engine := iguagile.New(factory, store)

	tlsConf, err := generateTLSConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err := engine.Start(context.Background(), address, apiAddr, tlsConf); err != nil {
		log.Fatal(err)
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
		Certificates:       []tls.Certificate{tlsCert},
		NextProtos:         []string{"iguagile-example"},
		InsecureSkipVerify: true,
	}, nil
}
