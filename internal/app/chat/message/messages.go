package message

import (
	"github.com/scirelli/httpd-go-test/internal/app/chat/user"
)

//Control message that client sends back. The message the client sends back can have any one of these signal messages.
type Control struct {
	User    *user.User
	Content content       `json:"content"`
	Create  createMessage `json:"create"`
	Error   errorMessage  `json:"error"`
}

type content struct {
	Text string `json:"text"`
}

type errorMessage struct {
	Error string `json:"error"`
}

type createMessage struct {
	UserName string `json:"username"`
}
