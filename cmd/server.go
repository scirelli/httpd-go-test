package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	gotime "time"

	gorwebsocket "github.com/gorilla/websocket"
	"github.com/scirelli/httpd-go-test/internal/pkg/time"
	"github.com/scirelli/httpd-go-test/internal/pkg/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var upgrader = gorwebsocket.Upgrader{}

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

func sendOnInterval(connectionManager *websocket.ConnectionManager) context.CancelFunc {
	ctx, cancel := context.WithTimeout(context.Background(), 100*gotime.Second)
	//defer cancel()
	go time.DoEvery(ctx, 2*gotime.Second, func(t gotime.Time) {
		log.Println("Sending a message")
		errChan := connectionManager.SendAll(strings.NewReader(fmt.Sprintf(`{"x": "%d", "y": %d}`, t.Second(), 1)))
		for err := range errChan {
			log.Println(err)
		}
	})

	return cancel
}

func main() {
	var connectionManager = websocket.ConnectionManager{
		Upgrader: gorwebsocket.Upgrader{},
	}

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
