package main

import (
	"github.com/iguagile/iguagile-engine/hub"
	"log"
	"net/http"
)

func main() {

	h := hub.NewHub()
	go h.Run()
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		hub.ServeWs(h, writer, request)
	})
	if err := http.ListenAndServe(":5000", nil); err != nil {
		log.Fatal(err)
	}
}
