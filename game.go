// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package billion

import (
	"bytes"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize: 50,
}

const MapSize = 32768
const ChunkSize = 32
const ChunkCount = MapSize / ChunkSize
const gameTickDuration = 100 * time.Millisecond

const gameStateUnitBits = 32

type GameState [MapSize][MapSize / gameStateUnitBits]uint32

type Chunk [ChunkSize][ChunkSize / gameStateUnitBits]uint32

func (gs *GameState) uncoverBox(x, y uint16) bool {
	if x >= MapSize || y >= MapSize {
		return false
	}

	mask := uint32(1 << ((gameStateUnitBits - 1) - x%(gameStateUnitBits-1)))
	old := atomic.OrUint32(&gs[y][x/gameStateUnitBits], mask)

	return old&mask == 0
}

func (gs *GameState) getChunk(chunkX, chunkY uint16) Chunk {
	var chunk Chunk
	if chunkX >= MapSize || chunkY >= MapSize {
		return chunk
	}

	for innerY := 0; innerY < len(chunk); innerY++ {
		for innerX := 0; innerX < len(chunk[innerY]); innerX++ {
			chunk[innerY][innerX] = gs[int(chunkY)*ChunkSize+innerY][int(chunkX)*ChunkSize+innerX]
		}
	}

	return chunk
}

// Game maintains the set of active clients and broadcasts messages to the
// clients.
type Game struct {
	// Registered clients.
	clients map[*Player]bool

	broadcast chan []byte

	// Register requests from the clients.
	register chan *Player

	// Unregister requests from clients.
	unregister chan *Player

	uncoveredCount atomic.Uint32

	recentUncoversMu *sync.Mutex
	recentUncovers   []Uncover
	lastTickTime     time.Time
	gameMap          GameState
}

func NewGame() *Game {
	return &Game{
		broadcast:        make(chan []byte),
		register:         make(chan *Player),
		unregister:       make(chan *Player),
		clients:          make(map[*Player]bool),
		recentUncoversMu: &sync.Mutex{},
	}
}

func (h *Game) Run() {
	go h.startSendingGameStats(gameTickDuration)
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case msg := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Game) startSendingGameStats(freq time.Duration) {
	ticker := time.NewTicker(freq)

	for {
		<-ticker.C
		gameStats := MessageGameStats{
			UncoveredCount:    h.uncoveredCount.Load(),
			OnlineCount:       uint32(len(h.clients)),
			RecentlyUncovered: h.getAndClearRecentUncovers(),
		}
		buf := &bytes.Buffer{}
		err := gameStats.Encode(buf)
		if err != nil {
			log.Println("error: ", err)
		}
		h.broadcast <- buf.Bytes()
	}
}

func (g *Game) getAndClearRecentUncovers() []Uncover {
	g.recentUncoversMu.Lock()
	defer g.recentUncoversMu.Unlock()

	uncovers := g.recentUncovers
	g.recentUncovers = nil
	g.lastTickTime = time.Now()

	return uncovers
}

func (g *Game) uncoverBox(c Coordinates) {
	hasChanged := g.gameMap.uncoverBox(c.X, c.Y)
	if !hasChanged {
		return
	}

	g.uncoveredCount.Add(1)

	g.recentUncoversMu.Lock()
	defer g.recentUncoversMu.Unlock()

	timeDiff := time.Now().UnixMilli() - g.lastTickTime.UnixMilli()

	uncover := Uncover{
		Coordinates: c,
		TickTiming:  uint8(timeDiff),
	}

	g.recentUncovers = append(g.recentUncovers, uncover)
}

func (g *Game) Serve(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := NewPlayer(g, conn)
	client.startServing()
}
