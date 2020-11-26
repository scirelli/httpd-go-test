package chat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	gorwebsocket "github.com/gorilla/websocket"

	"github.com/scirelli/httpd-go-test/internal/app/chat/client"
	"github.com/scirelli/httpd-go-test/internal/app/chat/message"
	"github.com/scirelli/httpd-go-test/internal/app/chat/user"
	"github.com/scirelli/httpd-go-test/internal/pkg/websocket"
)

//Room represents a chat room where users can chat.
type Room struct {
	Users    []*user.User
	mux      sync.Mutex
	Upgrader gorwebsocket.Upgrader
	log      *log.Logger
}

//NewRoom creates a new chat room.
func NewRoom() Room {
	return Room{
		Upgrader: gorwebsocket.Upgrader{},
		log:      log.New(os.Stderr, "CHAT", log.LstdFlags),
	}
}

//SendMessage used to send a message to a client.
func SendMessage(msg io.Reader, c client.Client) {
	if err := c.Connection().SendMessage(msg); err != nil {
		log.Println(err)
	}
}

//SetupNewUser setup a new user in the chat room.
func (c *Room) SetupNewUser(usr *user.User) {
	if err := c.AddUser(usr); err != nil {
		c.log.Println(err)
		return
	}

	go c.listenForMessages(usr)
}

func (c *Room) listenForMessages(usr *user.User) {
	for {
		_, r, err := usr.Connection().Connection.NextReader()
		if err != nil {
			usr.Connection().Close()
			c.log.Println("Connection closed.")
			break
		}
		c.processMessage(r, usr)
	}
}

func (c *Room) processMessage(r io.Reader, usr *user.User) {
	dec := json.NewDecoder(r)
	for dec.More() {
		var msg message.Control
		err := dec.Decode(&msg)
		if err != nil {
			c.log.Println(err)
			break
		}

		if msg.Content.Text != "" {
			c.Relay(strings.NewReader(msg.Content.Text), usr)
		}
	}
}

//AddUser used to add users to the user list.
func (c *Room) AddUser(usr *user.User) error {
	var conn *websocket.Connection
	defer c.log.Println("New user added.")

	for i, u := range c.Users {
		if conn = u.Connection(); !conn.Active() {
			c.mux.Lock()
			c.Users[i] = usr
			c.mux.Unlock()

			return nil
		}
	}

	c.mux.Lock()
	c.Users = append(c.Users, usr)
	c.mux.Unlock()

	return nil
}

//RemoveUser used to remove user.
func (c *Room) RemoveUser(usr *user.User) error {
	defer log.Println("User removed.")

	for _, u := range c.Users {
		if u == usr {
			usr.Connection().Close()
			return nil
		}
	}

	return errors.New("Not found")
}

//CloseConnections used to close all open user connections.
func (c *Room) CloseConnections() error {
	defer log.Println("Closed all clients")

	for _, usr := range c.Users {
		if conn := usr.Connection(); conn != nil {
			usr.Connection().Close()
		}
	}

	return nil
}

//SendAll used to send a message to all connected users.
func (c *Room) SendAll(r io.Reader) error {
	msg, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	for _, usr := range c.Users {
		go SendMessage(bytes.NewReader(msg), usr)
	}

	return nil
}

//Relay used to send a message to relay a message from one client to the rest.
func (c *Room) Relay(r io.Reader, sender *user.User) error {
	msg, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	for _, usr := range c.Users {
		if usr != sender {
			go SendMessage(bytes.NewReader(msg), usr)
		}
	}

	return nil
}

func (c *Room) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	conn, err := c.Upgrader.Upgrade(res, req, nil)
	if err != nil {
		c.log.Print("upgrade:", err)
		return
	}
	var user user.User = user.New(websocket.NewConnection(conn), fmt.Sprint(len(c.Users)))
	c.SetupNewUser(&user)
}
