package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"image/color"
)

const (
	screenWidth  = 1280
	screenHeight = 720
	playerSpeed  = 5
	bulletSpeed  = 3
)

var (
	gameFont font.Face = basicfont.Face7x13
)

//go:embed assets/d.png
var catImageBytes []byte

//go:embed assets/m.png
var mouseImageBytes []byte

type Player struct {
	ID          string  `json:"id"`
	X           float32 `json:"x"`
	Y           float32 `json:"y"`
	MoveCounter int     `json:"moveCounter"`
	Role        string  `json:"role"`
}

type Game struct {
	Players       map[string]*Player
	conn          *websocket.Conn
	timer     float64
	highScore int
	catImage    *ebiten.Image  // for cat (d.png)
	mouseImage  *ebiten.Image  // for others (m.png)
}

// Connect to WebSocket
func (g *Game) connectWebSocket() {
	u := url.URL{Scheme: "ws", Host: "10.40.134.115:8080", Path: "/ws"} // Replace with your server's IP
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
				Players   map[string]*Player `json:"players"`
				Timer     float64           `json:"timer"`
				HighScore int               `json:"highScore"`
			}
			if err := json.Unmarshal(message, &data); err != nil {
				log.Println("Unmarshal error:", err)
				continue
			}
			g.Players = data.Players
			g.timer = data.Timer
			g.highScore = data.HighScore
		}
	}()
}

func NewGame() *Game {
	// Load cat image (d.png)
	img1, _, err := image.Decode(bytes.NewReader(catImageBytes))
	if err != nil {
		log.Fatal("failed to load cat image:", err)
	}
	catImage := ebiten.NewImageFromImage(img1)

	// Load mouse image (m.png)
	img2, _, err := image.Decode(bytes.NewReader(mouseImageBytes))
	if err != nil {
		log.Fatal("failed to load mouse image:", err)
	}
	mouseImage := ebiten.NewImageFromImage(img2)

	game := &Game{
		Players:      make(map[string]*Player),
		timer:     0,
		highScore: 0,
		catImage:   catImage,
		mouseImage: mouseImage,
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
		op := &ebiten.DrawImageOptions{}
		
		// Choose image based on role
		var playerImage *ebiten.Image
		if player.Role == "cat" {
			playerImage = g.catImage
		} else {
			playerImage = g.mouseImage
		}
		
		// Get image dimensions
		w, h := playerImage.Bounds().Dx(), playerImage.Bounds().Dy()
		
		// Center the image on the player position
		op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
		
		// Scale the image if needed (adjust these values to change size)
		scale := 0.1 // Adjust this value to make image larger or smaller
		op.GeoM.Scale(scale, scale)
		
		// Move to player position
		op.GeoM.Translate(float64(player.X), float64(player.Y))
		
		screen.DrawImage(playerImage, op)
	}

	// Draw high score in top left
	highScoreText := fmt.Sprintf("Best Score: %d", g.highScore)
	text.Draw(screen, highScoreText, gameFont, 20, 20, color.Black)

	// Draw counter and timer in top right with adjusted positions
	counterText := fmt.Sprintf("Current Freak-ye Catches: %d", catCounter)
	text.Draw(screen, counterText, gameFont, screenWidth-250, 20, color.Black)
	
	timerText := fmt.Sprintf("Time until Freak-ye switch: %.1f", g.timer)
	text.Draw(screen, timerText, gameFont, screenWidth-250, 40, color.Black)
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
