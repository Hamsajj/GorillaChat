package client

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	ID         string
	ctx        context.Context
	serverURL  url.URL
	connection *websocket.Conn
	done       chan struct{}
}

func New(ctx context.Context, serverURL url.URL) (*Client, func() error) {
	id := uuid.NewString()
	c := &Client{id, ctx, serverURL, nil, make(chan struct{})}
	return c, c.cleanup
}

func (c *Client) Start() error {
	var err error
	c.connection, _, err = websocket.DefaultDialer.Dial(c.serverURL.String(), nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer c.connection.Close()

	go func() {
		defer close(c.done)
		for {
			_, message, err := c.connection.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return nil
		case t := <-ticker.C:
			err := c.connection.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return err
			}
		case <-c.ctx.Done():
			log.Println("context done")
			err := c.cleanup()
			if err != nil {
				return err
			}
			return nil
		}
	}
}

func (c *Client) cleanup() error {
	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := c.connection.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
	if err != nil {
		log.Println("write close:", err)
		return err
	}
	for {
		select {
		case <-c.done:
			return nil
		case <-time.After(time.Second):
			return nil
		}
	}
}
