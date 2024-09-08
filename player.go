// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package billion

import (
	"bytes"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Player is a middleman between the websocket connection and the hub.
type Player struct {
	game *Game

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

func NewPlayer(g *Game, conn *websocket.Conn) *Player {
	client := &Player{game: g, conn: conn, send: make(chan []byte, 10)}
	client.game.register <- client

	return client
}

func (c *Player) startServing() {
	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go c.writePump()
	go c.readPump()
}

func (c *Player) handleMessage(msg []byte) {
	// log.Println("received message", msg)
	buf := bytes.NewBuffer(msg)
	messageType, err := parseMessageType(buf)
	if err != nil {
		log.Println("cannot parse message type", err)
		return
	}
	switch messageType {
	case MessageTypeBoxCheckRequest:
		var checkRequest MessageBoxCheckRequest
		err := checkRequest.Decode(buf)
		if err != nil {
			log.Println("cannot parse message", err)
			return
		}
		c.game.uncoverBox(checkRequest.Coordinates)
	case MessageTypeChunkRequest:
		var chunkRequest MessageChunkRequest
		err := chunkRequest.Decode(buf)
		if err != nil {
			log.Println("cannot parse message", err)
			return
		}
		log.Println("Chunk Request", chunkRequest)
		var response MessageChunksResponse
		for _, r := range chunkRequest.Chunks {
			chunk := c.game.gameMap.getChunk(r.X, r.Y)
			response.Chunks = append(response.Chunks, ChunkResponse{
				Coordinates: r,
				Chunk:       chunk,
			})
		}
		buf := &bytes.Buffer{}
		err = response.Encode(buf)
		if err != nil {
			log.Println("cannot encode response", err)
			return
		}
		c.send <- buf.Bytes()
	default:
		log.Println("Unknown message type", messageType)

	}
}

// func (c *Player) handleBoxCheckRequest(r MessageBoxCheckRequest) {

// }

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Player) readPump() {
	defer func() {
		c.game.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// msg, err := parseMessage(messageReader)
		// if err != nil {
		// 	errMessage := fmt.Sprintf("error parsing message: %v", err)
		// 	log.Printf(errMessage)
		// 	msg, err := prepareMessage(MessageError{Message: errMessage})
		// 	if err != nil {
		// 		log.Fatalf("cannot prepare message: %v", err)
		// 	}
		// 	c.send <- msg
		// }

		c.handleMessage(message)
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Player) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
