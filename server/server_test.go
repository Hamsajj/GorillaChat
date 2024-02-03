package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gorilla/websocket"
)

func TestServerSuccess(t *testing.T) {
	t.Parallel()
	t.Run("should connect to the server and read no error", func(t *testing.T) {
		t.Parallel()
		s := newTestServer()
		defer s.Close()
		conn := createNewConnection(t, s, "client1")
		defer conn.Close()
		require.NotNil(t, conn)
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second*2)))
		// should timeout without receiving any error message
		_, _, err := conn.ReadMessage()
		require.ErrorContains(t, err, "i/o timeout")
	})

	t.Run("two clients should connect to the server and read broadcast messages", func(t *testing.T) {
		t.Parallel()
		s := newTestServer()
		defer s.Close()
		conn1 := createNewConnection(t, s, "client1")
		defer conn1.Close()
		require.NotNil(t, conn1)
		conn2 := createNewConnection(t, s, "client2")
		defer conn2.Close()
		require.NotNil(t, conn2)
		require.NoError(t, conn1.WriteMessage(websocket.TextMessage, []byte("hello")))
		requireReadMessage(t, conn2, broadcastMessage{Sender: "client1", Message: "hello"})

		require.NoError(t, conn2.WriteMessage(websocket.TextMessage, []byte("world")))
		requireReadMessage(t, conn1, broadcastMessage{Sender: "client2", Message: "world"})
	})
}

func TestServerErrors(t *testing.T) {
	t.Parallel()
	t.Run("should return error because clientID is required", func(t *testing.T) {
		t.Parallel()
		s := newTestServer()
		defer s.Close()
		conn := createNewConnection(t, s, "")
		defer conn.Close()
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("hello")))
		// Expect the httpServer to return ClientIDRequiredError.
		requireReadError(t, conn, NewClientIDRequiredError())
	})

	t.Run("should return error because message type is not supported", func(t *testing.T) {
		t.Parallel()
		s := newTestServer()
		defer s.Close()
		conn := createNewConnection(t, s, "client1")
		defer conn.Close()
		require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("hello")))
		// Expect the httpServer to return UnsupportedMessageType.
		requireReadError(t, conn, NewUnsupportedMessageType())
	})
}

func newTestServer() *httptest.Server {
	server := New()
	testServer := httptest.NewServer(http.HandlerFunc(server.connect))
	go server.broadcastMessages()
	return testServer
}

func createNewConnection(t *testing.T, httpServer *httptest.Server, clientID string) *websocket.Conn {
	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")
	wsURL = wsURL + "?clientID=" + clientID
	websocketConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	return websocketConn
}

func requireReadMessage(t *testing.T, conn *websocket.Conn, expectedMessage broadcastMessage) {
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second*2)))
	_, msg1, err := conn.ReadMessage()
	require.NoError(t, err)
	message, err := toBroadcastMessage(msg1)
	require.NoError(t, err)
	require.Equal(t, expectedMessage, message)
}

func requireReadError(t *testing.T, conn *websocket.Conn, expected ErrorResponse) {
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second*2)))
	mt, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, websocket.TextMessage, mt)
	errorResponse, err := toError(msg)
	require.NoError(t, err)
	require.Equal(t, expected, errorResponse)
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
