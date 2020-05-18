package main

import (
	"log"
	"net"
	"os"
	"strconv"

	"github.com/iguagile/iguagile-engine/iguagile"
)

func main() {
	factory := &iguagile.RelayServiceFactory{}
	address := os.Getenv("ROOM_HOST")
	if address == "" {
		address = "localhost:0"
	}

	store, err := iguagile.NewRedis(os.Getenv("REDIS_HOST"))
	if err != nil {
		log.Fatal(err)
	}

	server, err := iguagile.NewRoomServer(factory, store, address)
	if err != nil {
		log.Fatal(err)
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}

	port, err := strconv.Atoi(os.Getenv("GRPC_PORT"))
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(server.Run(listener, port))
}
