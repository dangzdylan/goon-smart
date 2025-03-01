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
	titleSize    = 2.0  // Scale factor for title text
	numberSize   = 1.5  // Scale factor for numbers
)

var (
	gameFont font.Face = basicfont.Face7x13
)

//go:embed assets/d.png
var catImageBytes []byte

//go:embed assets/m.png
var mouseImageBytes []byte

// Add color mapping
var colorMap = map[string]color.Color{
	"red":    color.RGBA{255, 0, 0, 255},
	"orange": color.RGBA{255, 127, 0, 255},
	"green":  color.RGBA{0, 255, 0, 255},
	"blue":   color.RGBA{0, 0, 255, 255},
	"indigo": color.RGBA{75, 0, 130, 255},
	"violet": color.RGBA{138, 43, 226, 255},
}

type Player struct {
	ID          string  `json:"id"`
	X           float32 `json:"x"`
	Y           float32 `json:"y"`
	MoveCounter int     `json:"moveCounter"`
	Role        string  `json:"role"`
	Color       string  `json:"color"`
}

type Game struct {
	Players         map[string]*Player
	conn            *websocket.Conn
	timer           float64
	highScore       int
	highScoreColor  string
	catImage        *ebiten.Image  // for cat (d.png)
	mouseImage      *ebiten.Image  // for others (m.png)
}

// Connect to WebSocket
func (g *Game) connectWebSocket() {
	u := url.URL{Scheme: "ws", Host: "10.41.135.80:8080", Path: "/ws"} // Replace with your server's IP
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
				Players         map[string]*Player `json:"players"`
				Timer           float64           `json:"timer"`
				HighScore       int               `json:"highScore"`
				HighScoreColor  string            `json:"highScoreColor"`
			}
			if err := json.Unmarshal(message, &data); err != nil {
				log.Println("Unmarshal error:", err)
				continue
			}
			g.Players = data.Players
			g.timer = data.Timer
			g.highScore = data.HighScore
			g.highScoreColor = data.HighScoreColor
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
		timer:      0,
		highScore:  0,
		highScoreColor: "",
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

	// Draw title at top middle
	titleText := "RUN FROM FREAK-YE"
	titleBounds := text.BoundString(gameFont, titleText)
	titleX := (screenWidth - titleBounds.Dx()*int(titleSize)) / 2
	
	// Create title options with scaling
	titleOpts := &ebiten.DrawImageOptions{}
	titleOpts.GeoM.Scale(titleSize, titleSize)
	titleOpts.GeoM.Translate(float64(titleX), 30)

	// Create temporary image for scaled text
	titleImg := ebiten.NewImage(titleBounds.Dx()+1, titleBounds.Dy()+1)
	text.Draw(titleImg, titleText, gameFont, 0, -titleBounds.Min.Y, color.Black)
	
	// Draw scaled title
	screen.DrawImage(titleImg, titleOpts)

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
		
		// Scale the image if needed
		scale := 0.1
		op.GeoM.Scale(scale, scale)
		
		// Move to player position
		op.GeoM.Translate(float64(player.X), float64(player.Y))
		
		screen.DrawImage(playerImage, op)

		// Draw colored dot in center using a small filled rectangle instead
		if playerColor, ok := colorMap[player.Color]; ok {
			dotSize := 5
			dotImg := ebiten.NewImage(dotSize*2, dotSize*2)
			dotImg.Fill(playerColor)
			
			dotOp := &ebiten.DrawImageOptions{}
			dotOp.GeoM.Translate(float64(player.X)-float64(dotSize), float64(player.Y)-float64(dotSize))
			screen.DrawImage(dotImg, dotOp)
		}
	}

	// Draw high score in top left with color and scaled number
	scoreText := "Best Score: "
	scoreNum := fmt.Sprintf("%d", g.highScore)
	
	scoreColor := color.RGBA{0, 0, 0, 255}  // Default black
	if g.highScoreColor != "" {
		if c, ok := colorMap[g.highScoreColor]; ok {
			if rgba, ok := c.(color.RGBA); ok {
				scoreColor = rgba
			}
		}
	}
	
	// Draw label normally
	text.Draw(screen, scoreText, gameFont, 20, 20, scoreColor)
	
	// Draw number scaled
	numBounds := text.BoundString(gameFont, scoreNum)
	numImg := ebiten.NewImage(numBounds.Dx()+1, numBounds.Dy()+1)
	text.Draw(numImg, scoreNum, gameFont, 0, -numBounds.Min.Y, scoreColor)
	
	numOpts := &ebiten.DrawImageOptions{}
	numOpts.GeoM.Scale(numberSize, numberSize)
	numOpts.GeoM.Translate(float64(20+text.BoundString(gameFont, scoreText).Dx()), 5)
	screen.DrawImage(numImg, numOpts)

	// Draw counter in top right with scaled number
	catchText := "Current Freak-ye Catches: "
	catchNum := fmt.Sprintf("%d", catCounter)
	
	// Draw label normally
	text.Draw(screen, catchText, gameFont, screenWidth-300, 20, color.Black)
	
	// Draw number scaled
	catchBounds := text.BoundString(gameFont, catchNum)
	catchImg := ebiten.NewImage(catchBounds.Dx()+1, catchBounds.Dy()+1)
	text.Draw(catchImg, catchNum, gameFont, 0, -catchBounds.Min.Y, color.Black)
	
	catchOpts := &ebiten.DrawImageOptions{}
	catchOpts.GeoM.Scale(numberSize, numberSize)
	catchOpts.GeoM.Translate(float64(screenWidth-300+text.BoundString(gameFont, catchText).Dx()), 5)
	screen.DrawImage(catchImg, catchOpts)

	// Draw timer with scaled number
	timerText := "Time until Freak-ye switch: "
	timerNum := fmt.Sprintf("%.1f", g.timer)
	
	// Draw label normally
	text.Draw(screen, timerText, gameFont, screenWidth-300, 40, color.Black)
	
	// Draw number scaled
	timerBounds := text.BoundString(gameFont, timerNum)
	timerImg := ebiten.NewImage(timerBounds.Dx()+1, timerBounds.Dy()+1)
	text.Draw(timerImg, timerNum, gameFont, 0, -timerBounds.Min.Y, color.Black)
	
	timerOpts := &ebiten.DrawImageOptions{}
	timerOpts.GeoM.Scale(numberSize, numberSize)
	timerOpts.GeoM.Translate(float64(screenWidth-300+text.BoundString(gameFont, timerText).Dx()), 25)
	screen.DrawImage(timerImg, timerOpts)
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
