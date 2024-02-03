package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type ClientID string

type broadcastMessage struct {
	Sender  ClientID `json:"sender"`
	Message string   `json:"message"`
}

type Server struct {
	clients       map[ClientID]*websocket.Conn
	broadcastChan chan broadcastMessage
	upgrader      websocket.Upgrader
}

func New() *Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // allow all connections
		},
	}
	return &Server{make(map[ClientID]*websocket.Conn), make(chan broadcastMessage), upgrader}
}

func (s *Server) Start(port int) error {
	http.HandleFunc("/connect", s.connect)
	log.Println("Starting httpServer on port", port)
	go s.broadcastMessages()
	//nolint:gosec // this httpserver is used for socket connection so should not have timeout
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (s *Server) connect(w http.ResponseWriter, r *http.Request) {
	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade connect:", err)
		return
	}
	defer c.Close()
	clientID := ClientID(r.URL.Query().Get("clientID"))
	if clientID == "" {
		s.writeJSON(c, NewClientIDRequiredError(), "")
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
			s.writeJSON(c, NewUnsupportedMessageType(), clientID)
		}
	}
}

// broadcastMessages reads messages from the broadcastChan and sends them to all connected clients.
func (s *Server) broadcastMessages() {
	for msg := range s.broadcastChan {
		for id, c := range s.clients {
			if id == msg.Sender {
				continue
			}
			s.writeJSON(c, msg, id)
		}
	}
}

func (s *Server) writeJSON(c *websocket.Conn, data interface{}, cid ClientID) {
	s.handleWriteError(c.WriteJSON(data), cid)
}

// unregisterClient removes the client with the given id from the httpServer's map of clients.
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
