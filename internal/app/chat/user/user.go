package user

import (
	"github.com/google/uuid"

	"github.com/scirelli/httpd-go-test/internal/pkg/websocket"
)

//User a user represents a user in the chat room.
type User struct {
	connection *websocket.Connection
	ID         uuid.UUID
	Name       string
	Channel    uuid.UUID
}

//New create a new user.
func New(connection *websocket.Connection, name string) User {
	return User{
		connection: connection,
		ID:         uuid.New(),
		Name:       name,
	}
}

//Connection implement the Client interface
func (u *User) Connection() *websocket.Connection {
	return u.connection
}

//SetConnection set the connection
func (u *User) SetConnection(connection *websocket.Connection) *User {
	u.connection = connection
	return u
}

//String implement Stringer
func (u *User) String() string {
	return u.Name
}
