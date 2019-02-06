package main

import (
	"github.com/labstack/echo"
	"gopkg.in/olahol/melody.v1"
	"net/http"
)

func main() {
	m := melody.New()
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		http.ServeFile(c.Response().Writer, c.Request(), "index.html")
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.GET("/ws", func(c echo.Context) error {
		m.HandleRequest(c.Response().Writer, c.Request())
		return nil
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		m.Broadcast(msg)
	})
	e.Start(":5000")
}
