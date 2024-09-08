package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jakubc-projects/billion"
)

var port = os.Getenv("PORT")

func main() {
	mux := http.NewServeMux()

	wsServer := billion.NewGame()
	go wsServer.Run()

	mux.HandleFunc("/websocket", wsServer.Serve)

	mux.Handle("/", http.FileServer(http.Dir("public")))

	addr := ":8080"
	if port != "" {
		addr = fmt.Sprintf(":%s", port)
	}

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
