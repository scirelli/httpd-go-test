package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

//WebSocketConnection represents a web socket connection.
type WebSocketConnection struct {
	Connection *websocket.Conn
	Active     bool
	mux        sync.Mutex
}

//SendMessage used to send a message to the listening client.
func (wsc *WebSocketConnection) SendMessage(r io.Reader) error {
	if !wsc.Active {
		return nil
	}

	w, err := wsc.Connection.NextWriter(websocket.TextMessage)
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

//Close close the webSocket and set this connection to inactive.
func (wsc *WebSocketConnection) Close() error {
	wsc.mux.Lock()
	wsc.Active = false
	wsc.Connection.Close()
	wsc.mux.Unlock()

	return nil
}

//ConnectionManager a tool for managing web socket connections.
type ConnectionManager struct {
	Connections []*WebSocketConnection
	mux         sync.Mutex
}

//AddConnection used to add connections to the connection list.
func (cm *ConnectionManager) AddConnection(webSocketConnection *WebSocketConnection) error {
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
func (cm *ConnectionManager) RemoveConnection(webSocketConnection *WebSocketConnection) error {
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

//SendMessage used to send a message to all connected clients.
func SendMessage(r io.Reader, sendTo []*WebSocketConnection) <-chan error {
	var wg sync.WaitGroup
	errChan := make(chan error)

	msg, err := ioutil.ReadAll(r)
	if err != nil {
		errChan <- err
		return errChan
	}

	wg.Add(len(sendTo))
	for _, conn := range sendTo {
		go (func(conn *WebSocketConnection, r io.Reader) {
			if err := conn.SendMessage(strings.NewReader(string(msg))); err != nil {
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

//SendAll used to send a message to all connected clients.
func (cm *ConnectionManager) SendAll(r io.Reader) <-chan error {
	return SendMessage(r, cm.Connections)
}

//Relay used to send a message to relay a message from one client to the rest.
func (cm *ConnectionManager) Relay(r io.Reader, sender *WebSocketConnection) <-chan error {
	var others []*WebSocketConnection

	for _, wsc := range cm.Connections {
		if wsc != sender {
			others = append(others, wsc)
		}
	}

	return SendMessage(r, others)
}

var addr = flag.String("addr", "localhost:8080", "http service address")
var upgrader = websocket.Upgrader{} // use default options

func (cm *ConnectionManager) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	c, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	var wsc = &WebSocketConnection{
		Connection: c,
		Active:     true,
	}

	go func(wsc *WebSocketConnection) {
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

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
}

func doEvery(ctx context.Context, d time.Duration, f func(time.Time)) error {
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case x := <-ticker.C:
			f(x)
		}
	}
}

func sendOnInterval(connectionManager *ConnectionManager) context.CancelFunc {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	//defer cancel()
	go doEvery(ctx, 2*time.Second, func(t time.Time) {
		log.Println("Sending a message")
		errChan := connectionManager.SendAll(strings.NewReader(fmt.Sprintf(`{"x": "%d", "y": %d}`, t.Second(), 1)))
		for err := range errChan {
			log.Println(err)
		}
	})

	return cancel
}

func main() {
	var connectionManager = ConnectionManager{}

	flag.Parse()
	log.SetFlags(0)

	http.HandleFunc("/echo", echo)
	http.HandleFunc("/home", home)
	http.Handle("/ws", &connectionManager)
	http.Handle("/", http.FileServer(http.Dir("./web/static")))

	//sendOnInterval(&connectionManager)

	log.Println("Listening on " + *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>  
window.addEventListener("load", function(evt) {

    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;

    var print = function(message) {
        var d = document.createElement("div");
        d.textContent = message;
        output.appendChild(d);
    };

    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("{{.}}");
        ws.onopen = function(evt) {
            print("OPEN");
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
            print("RESPONSE: " + evt.data);
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };

    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        print("SEND: " + input.value);
        ws.send(input.value);
        return false;
    };

    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };

});
</script>
</head>
	<body>
		<table>
			<tr>
				<td valign="top" width="50%">
					<p>
						Click "Open" to create a connection to the server, 
						"Send" to send a message to the server and "Close" to close the connection. 
						You can change the message and send multiple times.
					<p>
					<form>
						<button id="open">Open</button>
						<button id="close">Close</button>
						<p>
							<input id="input" type="text" value="Hello world!">
							<button id="send">Send</button>
						</p>
					</form>
				</td>
				<td valign="top" width="50%">
					<div id="output"></div>
				</td>
			</tr>
		</table>
	</body>
</html>
`))
