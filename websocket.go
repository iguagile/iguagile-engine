package main

import (
	"log"
	"net/http"

	"github.com/iguagile/iguagile-engine/hub"
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
