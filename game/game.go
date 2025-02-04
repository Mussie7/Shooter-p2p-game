package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"log"
	"image/color"
)

// Screen & player properties
const (
	ScreenWidth  = 800
	ScreenHeight = 600
	PlayerSize   = 20
	PlayerSpeed  = 2
)

// Player struct
type Player struct {
	x, y float64
}

// Game struct
type Game struct {
	player Player
}

// Update handles movement
func (g *Game) Update() error {
	// Movement logic
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.player.y -= PlayerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.player.y += PlayerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.player.x -= PlayerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.player.x += PlayerSpeed
	}

	// Prevent player from moving off-screen
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

	return nil
}

// Draw renders everything
func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DrawRect(screen, g.player.x, g.player.y, PlayerSize, PlayerSize, color.White)
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
