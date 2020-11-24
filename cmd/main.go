package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()
	setupRoutes(mux)

	errs := make(chan error)
	go listenHTTP(mux, errs)
	//go listenHTTPS(mux, errs)
	for i := range errs {
		log.Println(i)
	}
}

func listenHTTP(mux *http.ServeMux, errs chan<- error) {
	errs <- http.ListenAndServe(fmt.Sprintf(":%v", 8181), mux)
}

//func listenHTTPS(mux *http.ServeMux, errs chan<- error) {
//errs <- http.ListenAndServeTLS(fmt.Sprintf(":%v", 443, "your_full_chain.pem", "your_private_key.pem", mux))
//}
func setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/time", endpointTime)

	mux.Handle("/", http.FileServer(http.Dir("./web/static")))
}

func endpointTime(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", time.Now().Format("2006-01-02T15:04:05.999999-07:00"))
}
