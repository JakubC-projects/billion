package main

import (
	"log"
	"net/http"

	diamond "github.com/jakubc-projects/find-diamond"
)

func main() {
	mux := http.NewServeMux()

	wsServer := diamond.NewGame()
	go wsServer.Run()

	mux.HandleFunc("/websocket", wsServer.Serve)

	mux.Handle("/", http.FileServer(http.Dir("public")))

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

// func broadcastMessage(data []byte) error {
// 	fmt.Println("broadcasting", data)
// 	msg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, data)
// 	if err != nil {
// 		return err
// 	}
// 	for i, c := range clients {
// 		if c.conn
// 		err := c.conn.WritePreparedMessage(msg)
// 		if err != nil {
// 			c.conn.Close()
// 			clients = slices.Delete(clients, i, i+1)
// 			log.Println("client error", err)
// 		}
// 	}
// 	return nil
// }

// func websocketHandler(w http.ResponseWriter, r *http.Request) {
// 	conn, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		log.Println(err)
// 		return
// 	}

// 	clients = append(clients, Client{conn})
// 	go handleMessages(conn)
// }

// func handleMessages(conn *websocket.Conn) {
// 	for {
// 		_, buf, err := conn.ReadMessage()
// 		if err != nil {
// 			log.Println(err)
// 			return
// 		}
// 		err = broadcastMessage(buf)
// 		if err != nil {
// 			log.Println(err)
// 			return
// 		}
// 	}

// }
