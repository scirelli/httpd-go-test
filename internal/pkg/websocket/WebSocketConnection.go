package websocket

import (
	"io"
	"sync"

	gorwebsocket "github.com/gorilla/websocket"
)

//Connection represents a web socket connection.
type Connection struct {
	Connection *gorwebsocket.Conn
	Active     bool
	mux        sync.Mutex
}

//SendMessage used to send a message to the listening client.
func (wsc *Connection) SendMessage(r io.Reader) error {
	if !wsc.Active {
		return nil
	}

	w, err := wsc.Connection.NextWriter(gorwebsocket.TextMessage)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	return nil
}

//Close close the gorwebSocket and set this connection to inactive.
func (wsc *Connection) Close() error {
	wsc.mux.Lock()
	wsc.Active = false
	wsc.Connection.Close()
	wsc.mux.Unlock()

	return nil
}
