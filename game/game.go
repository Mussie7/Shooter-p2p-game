package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"log"
	"image/color"
)

// Screen size
const (
	ScreenWidth  = 800
	ScreenHeight = 600
	PlayerSize   = 20
)

// Player struct
type Player struct {
	x, y float64
}

// Game struct
type Game struct {
	player Player
}

// Update runs before every frame (Nothing to update yet)
func (g *Game) Update() error {
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
	// Create game instance with player centered
	game := &Game{
		player: Player{x: ScreenWidth/2 - PlayerSize/2, y: ScreenHeight/2 - PlayerSize/2},
	}

	// Set window size and title
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("2D Battle Royale")

	// Run the game
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
