package main

import (
	"log"
	"net"
	"net/http"

	"github.com/iguagile/iguagile-engine/iguagile"
)

func main() {
	room := iguagile.NewRoom()
	tcpAddr, err := net.ResolveTCPAddr("tcp", ":4000")
	if err != nil {
		log.Fatal(err)
	}
	listen, err := net.ListenTCP("tcp", tcpAddr)
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
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		iguagile.ServeWebsocket(room, writer, request)
	})
	if err := http.ListenAndServe(":5000", nil); err != nil {
		log.Fatal(err)
	}
}
