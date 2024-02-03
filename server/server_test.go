package server

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestServerSuccess(t *testing.T) {
	t.Parallel()
	t.Run("should connect to the server and read no error", func(t *testing.T) {
		h := newTestHandler()
		defer h.close()
		conn := h.createNewConnection(t, "client1")
		defer conn.Close()
		assert.NotNil(t, conn)
		assert.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second*2)))
		// should timeout without receiving any error message
		_, _, err := conn.ReadMessage()
		assert.ErrorContains(t, err, "i/o timeout")
	})

	t.Run("two clients should connect to the server and read broadcast messages", func(t *testing.T) {
		h := newTestHandler()
		defer h.close()
		conn1 := h.createNewConnection(t, "client1")
		defer conn1.Close()
		assert.NotNil(t, conn1)
		conn2 := h.createNewConnection(t, "client2")
		defer conn2.Close()
		assert.NotNil(t, conn2)
		assert.NoError(t, conn1.WriteMessage(websocket.TextMessage, []byte("hello")))
		assertToReadJSONMessage(t, conn2, broadcastMessage{Sender: "client1", Message: "hello"})

		assert.NoError(t, conn2.WriteMessage(websocket.TextMessage, []byte("world")))
		assertToReadJSONMessage(t, conn1, broadcastMessage{Sender: "client2", Message: "world"})

	})
}

func TestServerErrors(t *testing.T) {
	t.Parallel()
	t.Run("should return error because clientID is required", func(t *testing.T) {
		h := newTestHandler()
		conn := h.createNewConnection(t, "")
		defer conn.Close()
		assert.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("hello")))
		// Expect the httpServer to return ClientIDRequiredError.
		assertToReadError(t, conn, ClientIDRequiredError)
	})

	t.Run("should return error because message type is not supported", func(t *testing.T) {
		h := newTestHandler()
		conn := h.createNewConnection(t, "client1")
		defer conn.Close()
		assert.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("hello")))
		// Expect the httpServer to return UnsupportedMessageType.
		assertToReadError(t, conn, UnsupportedMessageType)
	})
}

type handler struct {
	wsServer   *Server
	httpServer *httptest.Server
}

func newTestHandler() *handler {
	server := New()
	testServer := httptest.NewServer(http.HandlerFunc(server.connect))
	go server.broadcastMessages()
	return &handler{server, testServer}
}

func (h *handler) createNewConnection(t *testing.T, clientID string) *websocket.Conn {
	wsURL := "ws" + strings.TrimPrefix(h.httpServer.URL, "http")
	wsURL = wsURL + "?clientID=" + clientID
	websocketConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	return websocketConn

}

func assertToReadJSONMessage(t *testing.T, conn *websocket.Conn, expectedMessage broadcastMessage) {
	assert.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second*2)))
	_, msg1, err := conn.ReadMessage()
	message, err := toBroadcastMessage(msg1)
	assert.NoError(t, err)
	assert.Equal(t, expectedMessage, message)

}

func assertToReadError(t *testing.T, conn *websocket.Conn, expected ErrorResponse) {
	assert.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second*2)))
	mt, msg, err := conn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, websocket.TextMessage, mt)
	errorResponse, err := toError(msg)
	assert.NoError(t, err)
	assert.Equal(t, expected, errorResponse)
}

func (h *handler) close() {
	h.httpServer.Close()
}

func toError(s []byte) (ErrorResponse, error) {
	data := ErrorResponse{}
	err := json.Unmarshal(s, &data)
	if err != nil {
		return ErrorResponse{}, err
	}
	return data, nil
}

func toBroadcastMessage(s []byte) (broadcastMessage, error) {
	data := broadcastMessage{}
	err := json.Unmarshal(s, &data)
	if err != nil {
		return broadcastMessage{}, err
	}
	return data, nil
}
