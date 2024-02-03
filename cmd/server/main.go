package main

import (
	"github.com/hamsajj/gorillachat/server"
	"log"
)

func main() {
	s := server.New()
	log.Fatal(s.Start(8080))
}
