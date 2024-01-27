package server

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all connections
	},
}

type ClientID string

type broadcastMessage struct {
	Sender  ClientID `json:"sender"`
	Message string   `json:"message"`
}

type Server struct {
	clients       map[ClientID]*websocket.Conn
	port          int
	broadcastChan chan broadcastMessage
}

func New(port int) *Server {
	return &Server{make(map[ClientID]*websocket.Conn), port, make(chan broadcastMessage)}
}

func (s *Server) Start() error {

	http.HandleFunc("/connect", s.connect)
	log.Println("Starting server on port", s.port)
	go s.broadcastMessages()
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)

}

func (s *Server) connect(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade connect:", err)
		return
	}
	defer c.Close()
	clientID := ClientID(r.URL.Query().Get("clientID"))
	if clientID == "" {
		s.writeJSON(c, ClientIDRequiredError, "")
	}

	s.clients[clientID] = c
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}
		switch mt {
		case websocket.TextMessage:
			s.broadcastChan <- broadcastMessage{clientID, string(message)}
		case websocket.CloseMessage:
			s.unregisterClient(clientID)
		default:
			s.writeJSON(c, UnsupportedMessageType, clientID)
		}
	}
}

// broadcastMessages reads messages from the broadcastChan and sends them to all connected clients.
func (s *Server) broadcastMessages() {
	for {
		select {
		case msg := <-s.broadcastChan:
			for id, c := range s.clients {
				if id == msg.Sender {
					continue
				}
				s.writeJSON(c, msg, id)
			}
		}
	}
}

func (s *Server) writeJSON(c *websocket.Conn, data interface{}, cid ClientID) {
	s.handleWriteError(c.WriteJSON(data), cid)
}

// unregisterClient removes the client with the given id from the server's map of clients.
func (s *Server) unregisterClient(id ClientID) {
	go func() {
		s.broadcastChan <- broadcastMessage{"", fmt.Sprintf("%s disconnected", id)}
		delete(s.clients, id)
	}()
}

// handleWriteError handles errors that occur when writing to a client's websocket connection
// if error is a *websocket.CloseError will unregister the client.
func (s *Server) handleWriteError(err error, clientID ClientID) {
	if err == nil {
		return
	}
	if connectionIsClosedError(err) {
		if clientID != "" {
			log.Println("unregistering client because of unexpected close error", clientID)
			s.unregisterClient(clientID)
		}
		err = fmt.Errorf("unexpected close error: %w", err)
	}
	if errors.Is(err, websocket.ErrCloseSent) {
		return
	}
	log.Println("write error:", err)
}

func connectionIsClosedError(err error) bool {
	return websocket.IsUnexpectedCloseError(err) || errors.Is(err, websocket.ErrCloseSent)
}
