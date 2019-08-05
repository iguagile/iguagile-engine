package main

import (
	"fmt"
	"github.com/iguagile/iguagile-engine/iguagile"
	"log"
	"net"
	"net/http"
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
	fmt.Println("ListenTCP")
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		for {
			conn, err := listen.AcceptTCP()
			if err != nil {
				log.Println(err)
				continue
			}
			iguagile.ServeTCP(room, conn)
		}
	}()
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		iguagile.ServeWebsocket(room, writer, request)
	})
	fmt.Println("ListenWebsocket")
	if err := http.ListenAndServe(":5000", nil); err != nil {
		log.Fatal(err)
	}
}
