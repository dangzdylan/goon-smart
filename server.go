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
	screenWidth   = 1280
	screenHeight  = 720
	playerSpeed   = 5
)

// Player structs
type Player struct {
	ID          string  `json:"id"`
	X           float32 `json:"x"`
	Y           float32 `json:"y"`
	MoveCounter int     `json:"moveCounter"`
	Role        string  `json:"role"` // "cat" or "mouse"
	Color       string  `json:"color"`
}

// Game State
type Game struct {
	Players         map[string]*Player `json:"players"`
	Timer           float64           `json:"timer"`
	HighScore       int               `json:"highScore"`
	HighScoreColor  string            `json:"highScoreColor"`
	mu              sync.Mutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var gameState = Game{
	Players:         make(map[string]*Player),
	Timer:           7.0,
	HighScore:       0,
	HighScoreColor:  "",
}

var connections = make(map[string]*websocket.Conn)
var mu sync.Mutex

// Add a map to track colliding pairs at package level
var collidingPairs = make(map[string]bool)

// Add a variable to track if cat exists
var catExists bool = false

// Add rainbow colors at package level
var rainbowColors = []string{
	"red",    // #FF0000
	"orange", // #FF7F00
	"yellow", // #FFFF00
	"green",  // #00FF00
	"blue",   // #0000FF
	"indigo", // #4B0082
	"violet", // #8B00FF
}

var usedColors = make(map[string]bool)

// Add function to get random position
func getRandomPosition() (float32, float32) {
	// Add padding of 50 pixels from edges
	padding := float32(50)
	x := padding + float32(time.Now().UnixNano()%int64(screenWidth-2*padding))
	y := padding + float32(time.Now().UnixNano()%int64(screenHeight-2*padding))
	return x, y
}

// Add function to get random unused color
func getRandomColor() string {
	availableColors := make([]string, 0)
	for _, color := range rainbowColors {
		if !usedColors[color] {
			availableColors = append(availableColors, color)
		}
	}
	
	if len(availableColors) == 0 {
		// If all colors are used, reset and start over
		usedColors = make(map[string]bool)
		return rainbowColors[0]
	}
	
	color := availableColors[time.Now().UnixNano()%int64(len(availableColors))]
	usedColors[color] = true
	return color
}

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

	// Get random spawn position
	x, y := getRandomPosition()

	player := &Player{
		ID:          playerID,
		X:           x,
		Y:           y,
		MoveCounter: 0,
		Role:        role,
		Color:       getRandomColor(),
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
			usedColors[player.Color] = false  // Free up the color
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
							// Reset mouse to random position
							p2.X, p2.Y = getRandomPosition()
						} else {
							p2.MoveCounter++
							// Reset mouse to random position
							p1.X, p1.Y = getRandomPosition()
						}
						collidingPairs[pairKey] = true
						gameState.Timer = 7.0  // Changed from 4.0 to 7.0
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

// Update swapCatRole to track high score holder's color
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
		// Update high score if current score is higher
		if currentCat.MoveCounter > gameState.HighScore {
			gameState.HighScore = currentCat.MoveCounter
			gameState.HighScoreColor = currentCat.Color  // Store the color of the high score holder
			fmt.Printf("New high score: %d by %s player!\n", gameState.HighScore, currentCat.Color)
		}

		// Choose random mouse to become new cat
		newCat := mice[time.Now().UnixNano()%int64(len(mice))]
		
		// Swap roles and reset counters
		currentCat.Role = "mouse"
		currentCat.MoveCounter = 0  // Reset old cat's counter
		newCat.Role = "cat"
		newCat.MoveCounter = 0      // Reset new cat's counter
		
		fmt.Printf("New cat is %s (was %s)\n", newCat.ID, currentCat.ID)
	}
}

// Update sendGameUpdates to use 7.0 seconds consistently
func sendGameUpdates() {
	for {
		time.Sleep(30 * time.Millisecond)

		checkCollisions()

		// Update timer
		gameState.mu.Lock()
		gameState.Timer -= 15.0 / 1000.0  // Convert 30ms to seconds
		if gameState.Timer <= 0 {
			gameState.Timer = 7.0  // Changed from 4.0 to 7.0
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
