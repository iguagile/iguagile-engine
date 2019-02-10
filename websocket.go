package main

import (
	"net/http"

	"github.com/iguagile/iguagile-engine/hub"

	"github.com/labstack/echo"
)

func main() {

	h := hub.NewHub()
	go h.Run()
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		http.ServeFile(c.Response().Writer, c.Request(), "index.html")
		return nil
	})

	e.GET("/ws", func(c echo.Context) error {
		hub.ServeWs(h, c.Response().Writer, c.Request())
		return nil
	})

	if err := e.Start(":5000"); err != nil {
		e.Logger.Fatal(err)
	}

}
