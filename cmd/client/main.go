package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"os/signal"

	"github.com/hamsajj/gorillachat/client"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	serverAddress := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/echo"}
	c, cleanup := client.New(context.Background(), serverAddress)

	defer func() {
		err := cleanup()
		if err != nil {
			log.Println("error cleanup:", err)
		}
	}()

	go func() {
		err := c.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the client
	<-interrupt
}
