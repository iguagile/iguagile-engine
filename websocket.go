package main

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/olahol/melody.v1"

)

func main() {
	r := gin.Default()
	m := melody.New()

	r.GET("/", func(c *gin.Context) {
		m.HandleRequest(c.Writer, c.Request)
	})

	m.HandleMessageBinary(func(session *melody.Session, bytes []byte) {
		m.BroadcastBinary(bytes)
	})

	r.Run(":5000")
}