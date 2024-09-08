package main

import (
	"log"
	"net/http"

	"github.com/jakubc-projects/billion"
)

func main() {
	mux := http.NewServeMux()

	wsServer := billion.NewGame()
	go wsServer.Run()

	mux.HandleFunc("/websocket", wsServer.Serve)

	mux.Handle("/", http.FileServer(http.Dir("public")))

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
