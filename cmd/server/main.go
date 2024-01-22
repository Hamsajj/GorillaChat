package main

import (
	"github.com/hamsajj/gorillachat/server"
	"log"
)

func main() {
	s := server.New(8080)
	log.Fatal(s.Start())
}
