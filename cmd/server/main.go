package main

import (
	"log"

	"github.com/hamsajj/gorillachat/server"
)

func main() {
	const port = 8080
	s := server.New()
	log.Fatal(s.Start(port))
}
