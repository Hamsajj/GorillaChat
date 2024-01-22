package server

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type Server struct {
	port int
}

func New(port int) *Server {
	return &Server{port}
}

func (s *Server) Start() error {
	http.HandleFunc("/echo", s.echo)
	log.Println("Starting server on port", s.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}

func (s *Server) echo(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // allow all connections
		},
	} // use default options
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade echo:", err)
		return
	}
	defer c.Close()

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}

}
