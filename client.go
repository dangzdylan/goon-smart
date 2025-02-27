package main

import (
	"encoding/json"
	"image/color"
	"log"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
	playerSpeed  = 5
	bulletSpeed  = 3
)

type Player struct {
	ID       string  `json:"id"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Cooldown int     `json:"cooldown"`
}

type Game struct {
	Players map[string]Player
	conn    *websocket.Conn
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

			var data Game
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
		Players: make(map[string]Player),
	}
	game.connectWebSocket()
	return game
}

func (g *Game) Update() error {
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

	for _, player := range g.Players {
		vector.DrawFilledCircle(screen, player.X, player.Y, 30, color.Black, true)
	}
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
