package client

import "github.com/scirelli/httpd-go-test/internal/pkg/websocket"

//Client represents a client connection
type Client interface {
	Connection() *websocket.Connection
}
