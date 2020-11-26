package websocket

import (
	"io"
	"sync"

	gorwebsocket "github.com/gorilla/websocket"
)

//Connection represents a web socket connection.
type Connection struct {
	Connection *gorwebsocket.Conn
	active     bool
	mux        sync.Mutex
}

//NewConnection create a new Connection
func NewConnection(conn *gorwebsocket.Conn) *Connection {
	return &Connection{
		Connection: conn,
		active:     true,
	}
}

//SendMessage used to send a message to the listening client. Locks so that messages are forced to be syncronis.
func (conn *Connection) SendMessage(r io.Reader) error {
	conn.mux.Lock()
	defer conn.mux.Unlock()

	if !conn.active {
		return nil
	}

	w, err := conn.Connection.NextWriter(gorwebsocket.TextMessage)
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
func (conn *Connection) Close() error {
	conn.mux.Lock()
	defer conn.mux.Unlock()

	conn.active = false
	conn.Connection.Close()
	conn.Connection = nil

	return nil
}

//Active getter, says if the channel is active (true) or not (false).
func (conn *Connection) Active() bool {
	conn.mux.Lock()
	defer conn.mux.Unlock()
	return conn.active
}

//SetActive setter, sets the active status.
func (conn *Connection) SetActive(active bool) *Connection {
	conn.mux.Lock()
	defer conn.mux.Unlock()
	conn.active = active
	return conn
}
