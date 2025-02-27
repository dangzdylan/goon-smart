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
)

// Player structs
type Player struct {
	ID          string  `json:"id"`
	X           float32 `json:"x"`
	Y           float32 `json:"y"`
	MoveCounter int     `json:"moveCounter"`
	Role        string  `json:"role"` // "cat" or "mouse"
}

// Game State
type Game struct {
	Players map[string]*Player `json:"players"`
	Timer   float64           `json:"timer"`
	mu      sync.Mutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var gameState = Game{
	Players: make(map[string]*Player),
	Timer:   4.0,
}

var connections = make(map[string]*websocket.Conn)
var mu sync.Mutex

// Add a map to track colliding pairs at package level
var collidingPairs = make(map[string]bool)

// Add a variable to track if cat exists
var catExists bool = false

// WebSocket Handler
func gameHandler(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	defer conn.Close()

	playerID := fmt.Sprintf("Player-%d", len(gameState.Players)+1)
	
	// Assign role based on if cat exists
	role := "mouse"
	if !catExists {
		role = "cat"
		catExists = true
	}

	player := &Player{
		ID:          playerID,
		X:           screenWidth / 2,
		Y:           screenHeight / 2,
		MoveCounter: 0,
		Role:        role,
	}

	mu.Lock()
	gameState.Players[playerID] = player
	connections[playerID] = conn
	mu.Unlock()

	fmt.Printf("%s joined the game as %s!\n", playerID, role)

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
			player := gameState.Players[playerID]
			if player.Role == "cat" {
				catExists = false
			}
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
				// Calculate new position
				newX := player.X + float32(action.X*playerSpeed)
				newY := player.Y + float32(action.Y*playerSpeed)
				
				// Apply boundary checks
				if newX >= 0 && newX <= screenWidth {
					player.X = newX
				}
				if newY >= 0 && newY <= screenHeight {
					player.Y = newY
				}
			}
		}
		mu.Unlock()
	}
}

// Update collision detection to handle cat-mouse collisions
func checkCollisions() {
	gameState.mu.Lock()
	defer gameState.mu.Unlock()

	// Check each pair of players for collision
	players := make([]*Player, 0, len(gameState.Players))
	for _, p := range gameState.Players {
		players = append(players, p)
	}

	// Track current collisions to clean up old pairs
	currentCollisions := make(map[string]bool)

	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			p1 := players[i]
			p2 := players[j]

			// Only check collisions if one is cat and one is mouse
			if (p1.Role == "cat" && p2.Role == "mouse") || (p1.Role == "mouse" && p2.Role == "cat") {
				// Create a unique key for this pair
				pairKey := fmt.Sprintf("%s-%s", p1.ID, p2.ID)
				if p1.ID > p2.ID {
					pairKey = fmt.Sprintf("%s-%s", p2.ID, p1.ID)
				}

				// Calculate distance between players
				dx := p1.X - p2.X
				dy := p1.Y - p2.Y
				distance := float32(math.Sqrt(float64(dx*dx + dy*dy)))

				// If they're colliding
				if distance < 60 {
					fmt.Println("Collision detected between", p1.ID, "and", p2.ID)
					currentCollisions[pairKey] = true
					// Only increment if this is a new collision
					if !collidingPairs[pairKey] {
						// Find the cat and increment its counter
						if p1.Role == "cat" {
							p1.MoveCounter++
							// Reset mouse position
							p2.X = screenWidth / 2
							p2.Y = screenHeight / 2
						} else {
							p2.MoveCounter++
							// Reset mouse position
							p1.X = screenWidth / 2
							p1.Y = screenHeight / 2
						}
						collidingPairs[pairKey] = true
						gameState.Timer = 4.0  // Reset timer on collision
					}
				}
			}
		}
	}

	// Clean up collision tracking for pairs that are no longer colliding
	for pair := range collidingPairs {
		if !currentCollisions[pair] {
			delete(collidingPairs, pair)
		}
	}
}

// Add function to swap cat role
func swapCatRole() {
	gameState.mu.Lock()
	defer gameState.mu.Unlock()

	// Find current cat and all mice
	var currentCat *Player
	mice := make([]*Player, 0)
	
	for _, player := range gameState.Players {
		if player.Role == "cat" {
			currentCat = player
		} else {
			mice = append(mice, player)
		}
	}

	// Only proceed if we have both a cat and at least one mouse
	if currentCat != nil && len(mice) > 0 {
		// Choose random mouse to become new cat
		newCat := mice[time.Now().UnixNano()%int64(len(mice))]
		
		// Swap roles
		currentCat.Role = "mouse"
		newCat.Role = "cat"
		
		fmt.Printf("New cat is %s (was %s)\n", newCat.ID, currentCat.ID)
	}
}

// Update sendGameUpdates to handle cat rotation
func sendGameUpdates() {
	for {
		time.Sleep(30 * time.Millisecond)

		checkCollisions()

		// Update timer
		gameState.mu.Lock()
		gameState.Timer -= 15.0 / 1000.0  // Convert 30ms to seconds
		if gameState.Timer <= 0 {
			gameState.Timer = 4.0
			gameState.mu.Unlock()
			swapCatRole()  // Swap cat role when timer hits 0
		} else {
			gameState.mu.Unlock()
		}
		
		// Send updated state to all clients
		gameState.mu.Lock()
		stateJSON, _ := json.Marshal(gameState)
		for _, conn := range connections {
			conn.WriteMessage(websocket.TextMessage, stateJSON)
		}
		gameState.mu.Unlock()
	}
}

func main() {
	http.HandleFunc("/ws", gameHandler)
	fmt.Println("Multiplayer server running at ws://0.0.0.0:8080/ws")
	http.ListenAndServe("0.0.0.0:8080", nil)
}
