package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/jakubc-projects/billion"
)

var port = os.Getenv("PORT")

//go:embed public/*
var publicFiles embed.FS

func main() {
	mux := http.NewServeMux()

	wsServer := billion.NewGame()
	go wsServer.Run()

	mux.HandleFunc("/websocket", wsServer.Serve)

	fs, _ := fs.Sub(publicFiles, "public")
	mux.Handle("/", http.FileServerFS(fs))

	addr := ":8080"
	if port != "" {
		addr = fmt.Sprintf(":%s", port)
	}

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
