package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/scirelli/httpd-go-test/internal/app/chat"
)

var addr = flag.String("addr", "localhost:8181", "http service address")

func main() {
	var room1 = chat.NewRoom()

	flag.Parse()
	log.SetFlags(0)

	http.Handle("/room/1", &room1)
	http.Handle("/", http.FileServer(http.Dir("./web/static")))

	log.Println("Listening on http://" + *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
