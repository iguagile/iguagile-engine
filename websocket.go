package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"gopkg.in/olahol/melody.v1"
)

func main() {
	r := gin.Default()
	m := melody.New()

	r.GET("/", func(c *gin.Context) {
		if err := m.HandleRequest(c.Writer, c.Request); err != nil {
			log.Fatal(err)
		}

	})

	m.HandleMessageBinary(func(session *melody.Session, bytes []byte) {
		if err := m.BroadcastBinary(bytes); err != nil {
			log.Fatal(err)
		}
	})

	if err := r.Run(":5000"); err != nil {
		log.Fatal(err)
	}
}
