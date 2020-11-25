package websocket

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	gorwebsocket "github.com/gorilla/websocket"
)

//SendMessage used to send a message to all connected clients.
func SendMessage(r io.Reader, sendTo []*Connection) <-chan error {
	var wg sync.WaitGroup
	errChan := make(chan error)

	msg, err := ioutil.ReadAll(r)
	if err != nil {
		errChan <- err
		return errChan
	}

	wg.Add(len(sendTo))
	for _, conn := range sendTo {
		go (func(conn *Connection, r io.Reader) {
			if err := conn.SendMessage(bytes.NewReader(msg)); err != nil {
				errChan <- err
			}
			wg.Done()
		})(conn, r)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	return errChan
}

//ConnectionManager a tool for managing web socket connections.
type ConnectionManager struct {
	Connections []*Connection
	mux         sync.Mutex
	Upgrader    gorwebsocket.Upgrader
}

//AddConnection used to add connections to the connection list.
func (cm *ConnectionManager) AddConnection(webSocketConnection *Connection) error {
	for i, conn := range cm.Connections {
		conn.mux.Lock()
		if !conn.Active {
			conn.mux.Unlock()

			cm.mux.Lock()
			cm.Connections[i] = webSocketConnection
			cm.mux.Unlock()

			return nil
		}
		conn.mux.Unlock()
	}

	cm.mux.Lock()
	cm.Connections = append(cm.Connections, webSocketConnection)
	cm.mux.Unlock()

	return nil
}

//RemoveConnection used to remove client web socket connections.
func (cm *ConnectionManager) RemoveConnection(webSocketConnection *Connection) error {
	for _, conn := range cm.Connections {
		if conn == webSocketConnection {
			conn.Close()
			return nil
		}
	}

	return errors.New("Not found")
}

//CloseConnections used to close all open connections.
func (cm *ConnectionManager) CloseConnections() error {
	for _, conn := range cm.Connections {
		conn.Close()
	}

	return nil
}

//SendAll used to send a message to all connected clients.
func (cm *ConnectionManager) SendAll(r io.Reader) <-chan error {
	return SendMessage(r, cm.Connections)
}

//Relay used to send a message to relay a message from one client to the rest.
func (cm *ConnectionManager) Relay(r io.Reader, sender *Connection) <-chan error {
	var others []*Connection

	for _, wsc := range cm.Connections {
		if wsc != sender {
			others = append(others, wsc)
		}
	}

	return SendMessage(r, others)
}

func (cm *ConnectionManager) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	c, err := cm.Upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	var wsc = &Connection{
		Connection: c,
		Active:     true,
	}

	go func(wsc *Connection) {
		for {
			_, r, err := wsc.Connection.NextReader()
			if err != nil {
				wsc.Close()
				break
			}
			cm.Relay(r, wsc)
		}
	}(wsc)

	cm.AddConnection(wsc)
}
