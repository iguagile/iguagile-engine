package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/iguagile/iguagile-engine/iguagile"
)

const (
	roomPort = 4000
	apiPort  = 5000
)

func run() error {
	store, err := iguagile.NewRedis(os.Getenv("REDIS_HOST"))
	if err != nil {
		return err
	}

	server, err := iguagile.NewRoomServer(store, roomPort)
	if err != nil {
		return err
	}

	roomListener, err := net.Listen("tcp", fmt.Sprintf(":%v", roomPort))
	if err != nil {
		return err
	}

	return server.Run(roomListener, apiPort)
}

func main() {
	log.Fatal(run())
}
