package main

import (
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Screen & player properties
const (
	ScreenWidth  = 800
	ScreenHeight = 600
	PlayerSize   = 20
	PlayerSpeed  = 2
	LineLength   = 15 // Length of the direction indicator
)

// Player struct
type Player struct {
	x, y   float64
	angle  float64 // Player's facing angle
}

// Game struct
type Game struct {
	player Player
}

// Update handles movement & direction
func (g *Game) Update() error {
	vx, vy := 0.0, 0.0 // Velocity

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		vy -= PlayerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		vy += PlayerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		vx -= PlayerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		vx += PlayerSpeed
	}

	// Update position
	g.player.x += vx
	g.player.y += vy

	// Prevent moving off-screen
	if g.player.x < 0 {
		g.player.x = 0
	}
	if g.player.x > ScreenWidth-PlayerSize {
		g.player.x = ScreenWidth - PlayerSize
	}
	if g.player.y < 0 {
		g.player.y = 0
	}
	if g.player.y > ScreenHeight-PlayerSize {
		g.player.y = ScreenHeight - PlayerSize
	}

	// Update facing angle if moving
	if vx != 0 || vy != 0 {
		g.player.angle = math.Atan2(vy, vx) // Calculate angle
	}

	return nil
}

// Draw renders everything
func (g *Game) Draw(screen *ebiten.Image) {
	// Draw the player
	ebitenutil.DrawRect(screen, g.player.x, g.player.y, PlayerSize, PlayerSize, color.White)

	// Calculate the center of the player
	centerX := g.player.x + PlayerSize/2
	centerY := g.player.y + PlayerSize/2

	// Calculate the end point of the red line
	endX := centerX + math.Cos(g.player.angle)*LineLength
	endY := centerY + math.Sin(g.player.angle)*LineLength

	// Draw the direction indicator from the center
	ebitenutil.DrawLine(screen, centerX, centerY, endX, endY, color.RGBA{255, 0, 0, 255})
}

// Layout defines the screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func main() {
	// Initialize game with player at the center
	game := &Game{
		player: Player{x: ScreenWidth/2 - PlayerSize/2, y: ScreenHeight/2 - PlayerSize/2},
	}

	// Set window properties
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("2D Battle Royale")

	// Run game loop
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}