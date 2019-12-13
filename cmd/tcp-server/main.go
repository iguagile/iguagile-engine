package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/iguagile/iguagile-engine/iguagile"
)

const (
	roomPort = 4000
	apiPort  = 5000
)

func getIP() (string, error) {
	response, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}

	ip, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(ip), nil
}

func run() error {
	store, err := iguagile.NewRedis(os.Getenv("REDIS_HOST"))
	if err != nil {
		return err
	}

	roomIP, err := getIP()
	if err != nil {
		return err
	}

	roomHost := fmt.Sprintf("%v:%v", roomIP, roomPort)

	server, err := iguagile.NewRoomServer(store, roomHost)
	if err != nil {
		return err
	}

	roomListener, err := net.Listen("tcp", fmt.Sprintf(":%v", roomPort))
	if err != nil {
		return err
	}

	return server.Run(roomListener, apiPort)
}

func main() {
	log.Fatal(run())
}
