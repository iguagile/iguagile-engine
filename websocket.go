package main

import (
	"net/http"

	"github.com/labstack/gommon/log"

	"github.com/labstack/echo"
	"gopkg.in/olahol/melody.v1"
)

func main() {
	logger := log.Logger{}
	m := melody.New()
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		http.ServeFile(c.Response().Writer, c.Request(), "index.html")
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.GET("/ws", func(c echo.Context) error {
		if err := m.HandleRequest(c.Response().Writer, c.Request()); err != nil {
			logger.Fatal()
		}
		return nil
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		if err := m.Broadcast(msg); err != nil {
			logger.Fatal(err)
		}

	})
	if err := e.Start(":5000"); err != nil {

		logger.Fatal(err)

	}
}
