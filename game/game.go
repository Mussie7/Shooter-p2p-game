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
	LineLength   = 15  // Length of direction indicator
	BulletSize   = 5   // Bullet dimensions
	BulletSpeed  = 4   // Bullet movement speed
	ShotCooldown = 20  // Frames between shots
)

// Player struct
type Player struct {
	x, y   float64
	angle  float64
	cooldown int // Frames left until next shot
}

// Bullet struct
type Bullet struct {
	x, y   float64
	vx, vy float64
	active bool
	owner  *Player // Tracks who fired the bullet
}

// Game struct
type Game struct {
	player  Player
	bullets []Bullet
}

// Update handles movement, shooting, and collisions
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
		g.player.angle = math.Atan2(vy, vx)
	}

	// Shooting Mechanism
	if g.player.cooldown > 0 {
		g.player.cooldown--
	}
	if ebiten.IsKeyPressed(ebiten.KeySpace) && g.player.cooldown == 0 {
		g.shootBullet()
		g.player.cooldown = ShotCooldown
	}

	// Update bullets & check collisions
	for i := range g.bullets {
		if g.bullets[i].active {
			g.bullets[i].x += g.bullets[i].vx
			g.bullets[i].y += g.bullets[i].vy

			// Check if bullet is off-screen
			if g.bullets[i].x < 0 || g.bullets[i].x > ScreenWidth || g.bullets[i].y < 0 || g.bullets[i].y > ScreenHeight {
				g.bullets[i].active = false
				continue
			}

			// Fix: Player can't hit themselves
			if g.bullets[i].owner != &g.player && checkCollision(g.bullets[i], g.player) {
				g.bullets[i].active = false
				println("Bullet hit the player!")
			}
		}
	}

	return nil
}

// 🚀 **Bullet Collision Check**
func checkCollision(b Bullet, p Player) bool {
	return b.x > p.x && b.x < p.x+PlayerSize && b.y > p.y && b.y < p.y+PlayerSize
}

// Shoot a bullet
func (g *Game) shootBullet() {
	vx := BulletSpeed * math.Cos(g.player.angle)
	vy := BulletSpeed * math.Sin(g.player.angle)
	g.bullets = append(g.bullets, Bullet{
		x: g.player.x + PlayerSize/2, 
		y: g.player.y + PlayerSize/2, 
		vx: vx, vy: vy, 
		active: true,
		owner: &g.player, // Tracks the shooter
	})
}

// Draw renders everything
func (g *Game) Draw(screen *ebiten.Image) {
	// Draw the player
	ebitenutil.DrawRect(screen, g.player.x, g.player.y, PlayerSize, PlayerSize, color.White)

	// Draw facing direction
	centerX := g.player.x + PlayerSize/2
	centerY := g.player.y + PlayerSize/2
	endX := centerX + math.Cos(g.player.angle)*LineLength
	endY := centerY + math.Sin(g.player.angle)*LineLength
	ebitenutil.DrawLine(screen, centerX, centerY, endX, endY, color.RGBA{255, 0, 0, 255})

	// Draw bullets
	for _, b := range g.bullets {
		if b.active {
			ebitenutil.DrawRect(screen, b.x, b.y, BulletSize, BulletSize, color.RGBA{255, 255, 0, 255})
		}
	}
}

// Layout defines the screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func main() {
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