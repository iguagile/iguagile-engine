package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"

	"github.com/iguagile/iguagile-engine/client"
)

func main() {
	c := client.New()
	logger := log.New(os.Stdout, "iguagile-client-sample ", log.Lshortfile)
	c.SetLogger(logger)
	c.AddReceivedBinaryFunc(func(senderID int, data []byte) error {
		fmt.Printf("%04d:\t%v\n", senderID, string(data))
		return nil
	})

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"iguagile"},
	}

	ctx := context.Background()
	if err := c.DialQUIC(ctx, "localhost:5100", tlsConfig); err != nil {
		log.Fatal(err)
	}

	c.Run(ctx)
	log.Println("connected")

	stdin := bufio.NewScanner(os.Stdin)
	for stdin.Scan() {
		text := stdin.Text()
		c.SendBinary(client.AllClients, []byte(text))
	}
}
