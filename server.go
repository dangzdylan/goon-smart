package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Constants
const (
	screenWidth   = 640
	screenHeight  = 480
	playerSpeed   = 5
	bulletSpeed   = 3
	shootCooldown = 50
)

// Player & Bullet structs
type Player struct {
	ID       string  `json:"id"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Cooldown int     `json:"cooldown"`
}

type Bullet struct {
	X  float32 `json:"x"`
	Y  float32 `json:"y"`
	Dx float32 `json:"dx"`
	Dy float32 `json:"dy"`
}

// Game State
type Game struct {
	Players map[string]*Player `json:"players"`
	Bullets []Bullet           `json:"bullets"`
	mu      sync.Mutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var gameState = Game{
	Players: make(map[string]*Player),
	Bullets: []Bullet{},
}

var connections = make(map[string]*websocket.Conn)
var mu sync.Mutex

// WebSocket Handler
func gameHandler(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	defer conn.Close()

	playerID := fmt.Sprintf("Player-%d", len(gameState.Players)+1)
	player := &Player{ID: playerID, X: screenWidth / 2, Y: screenHeight / 2, Cooldown: 0}

	mu.Lock()
	gameState.Players[playerID] = player
	connections[playerID] = conn
	mu.Unlock()

	fmt.Println(playerID, "joined the game!")

	go handlePlayerInput(conn, playerID)
	go sendGameUpdates()

	select {}
}

// Handle Player Input
func handlePlayerInput(conn *websocket.Conn, playerID string) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			mu.Lock()
			delete(gameState.Players, playerID)
			delete(connections, playerID)
			mu.Unlock()
			fmt.Println(playerID, "disconnected")
			return
		}

		var action struct {
			Type string `json:"type"`
			X    int    `json:"x"`
			Y    int    `json:"y"`
		}
		json.Unmarshal(msg, &action)

		mu.Lock()
		player, exists := gameState.Players[playerID]
		if exists {
			switch action.Type {
			case "move":
				player.X += float32(action.X * playerSpeed)
				player.Y += float32(action.Y * playerSpeed)
			case "shoot":
				if player.Cooldown == 0 {
					mouseX := float32(action.X)
					mouseY := float32(action.Y)
					dx := mouseX - player.X
					dy := mouseY - player.Y
					length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
					if length != 0 {
						dx = dx / length * bulletSpeed
						dy = dy / length * bulletSpeed
						gameState.Bullets = append(gameState.Bullets, Bullet{player.X, player.Y, dx, dy})
						player.Cooldown = shootCooldown
					}
				}
			}
		}
		mu.Unlock()
	}
}

// Send Game Updates to Clients
func sendGameUpdates() {
	for {
		time.Sleep(30 * time.Millisecond)

		mu.Lock()
		stateJSON, _ := json.Marshal(gameState)

		for _, conn := range connections {
			conn.WriteMessage(websocket.TextMessage, stateJSON)
		}
		mu.Unlock()
	}
}

func main() {
	http.HandleFunc("/ws", gameHandler)
	fmt.Println("Multiplayer server running at ws://0.0.0.0:8080/ws")
	http.ListenAndServe("0.0.0.0:8080", nil)
}
