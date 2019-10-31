package main

import (
	"log"
	"net"

	"github.com/iguagile/iguagile-engine/iguagile"
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
	room := iguagile.NewRoom(serverID, store)
	tcpAddr, err := net.ResolveTCPAddr("tcp", ":4000")
	if err != nil {
		log.Fatal(err)
	}

	listen, err := net.ListenTCP("tcp", tcpAddr)
	log.Println("ListenTCP")
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listen.AcceptTCP()
		if err != nil {
			log.Println(err)
			continue
		}
		room.Serve(conn)
	}
}
