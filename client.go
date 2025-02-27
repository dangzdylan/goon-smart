package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

const (
	screenWidth  = 640
	screenHeight = 480
	playerSpeed  = 5
	bulletSpeed  = 3
)

var (
	gameFont font.Face = basicfont.Face7x13
)

type Player struct {
	ID          string  `json:"id"`
	X           float32 `json:"x"`
	Y           float32 `json:"y"`
	MoveCounter int     `json:"moveCounter"`
	Role        string  `json:"role"`
}

type Game struct {
	Players map[string]*Player
	conn    *websocket.Conn
	timer   float64
}

// Connect to WebSocket
func (g *Game) connectWebSocket() {
	u := url.URL{Scheme: "ws", Host: "10.0.0.109:8080", Path: "/ws"} // Replace with your server's IP
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("WebSocket connection failed:", err)
	}
	g.conn = conn

	// Listen for game state updates
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				return
			}

			var data struct {
				Players map[string]*Player `json:"players"`
			}
			if err := json.Unmarshal(message, &data); err != nil {
				log.Println("Unmarshal error:", err)
				continue
			}
			g.Players = data.Players
		}
	}()
}

func NewGame() *Game {
	game := &Game{
		Players: make(map[string]*Player),
		timer:   4.0,
	}
	game.connectWebSocket()
	return game
}

func (g *Game) Update() error {
	g.timer -= 1.0 / 60.0
	if g.timer <= 0 {
		g.timer = 4.0
	}

	dx, dy := 0, 0
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		dy = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		dy = 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		dx = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		dx = 1
	}

	if dx != 0 || dy != 0 {
		g.conn.WriteJSON(map[string]interface{}{
			"type": "move",
			"x":    dx,
			"y":    dy,
		})
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)

	// Find cat's counter to display
	var catCounter int
	for _, player := range g.Players {
		if player.Role == "cat" {
			catCounter = player.MoveCounter
			break
		}
	}

	// Draw players
	for _, player := range g.Players {
		// Choose color based on role
		var playerColor color.Color
		if player.Role == "cat" {
			playerColor = color.RGBA{R: 255, G: 0, B: 0, A: 255} // Red for cat
		} else {
			playerColor = color.RGBA{R: 100, G: 100, B: 100, A: 255} // Gray for mouse
		}

		vector.DrawFilledCircle(screen, player.X, player.Y, 30, playerColor, true)
	}

	// Draw counter and timer in top right
	counterText := fmt.Sprintf("Catches: %d", catCounter)
	text.Draw(screen, counterText, gameFont, screenWidth-120, 20, color.Black)
	
	timerText := fmt.Sprintf("Time: %.1f", g.timer)
	text.Draw(screen, timerText, gameFont, screenWidth-120, 40, color.Black)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Multiplayer Game")
	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
