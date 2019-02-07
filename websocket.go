package main

import (
	"net/http"

	"golang.org/x/net/websocket"

	"github.com/labstack/echo"
)

func main() {
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		http.ServeFile(c.Response().Writer, c.Request(), "index.html")
		return nil
	})

	e.GET("/ws", func(c echo.Context) error {
		websocket.Handler(func(ws *websocket.Conn) {
			defer func() {
				if err := ws.Close(); err != nil {
					c.Logger().Error(err)
				}
			}()
			for {
				// Read
				msg := ""
				err := websocket.Message.Receive(ws, &msg)
				if err != nil {
					c.Logger().Warn(err)
				}

				err = websocket.Message.Send(ws, msg)
				if err != nil {
					c.Logger().Warn(err)
				}
			}
		}).ServeHTTP(c.Response(), c.Request())
		return nil
	})

	if err := e.Start(":5000"); err != nil {
		e.Logger.Fatal(err)
	}
}
