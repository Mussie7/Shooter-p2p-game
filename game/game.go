package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"
	
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Screen & player properties
const (
	ScreenWidth     = 800
	ScreenHeight    = 600
	PlayerSize      = 20
	PlayerSpeed     = 2
	LineLength      = 15  // Length of direction indicator
	BulletSize      = 5   // Bullet dimensions
	BulletSpeed     = 4   // Bullet movement speed
	ShotCooldown    = 20  // Frames between shots
	DamageAmount    = 5   // New: Damage per bullet hit
	MaxHealth       = 100 // New: Maximum player health
	HealthBarWidth  = 20  // New: Health bar width
	HealthBarHeight = 3   // New: Health bar height
)

// Player struct (now includes ID)
type Player struct {
	id       string  // Unique player ID
	x, y     float64 // Position
	angle    float64 // Facing direction
	health   int     // Health bar
	cooldown int     // Shooting cooldown
}

// Bullet struct (tracks owner)
type Bullet struct {
	x, y     float64
	vx, vy   float64
	active   bool
	ownerID  string // ID of the player who fired it
}

// Game struct (supports multiple players)
type Game struct {
	players       map[string]*Player // Stores all players
	bullets       []Bullet           // Stores all bullets
	localPlayerID string             // ID of the local player
}

func (g *Game) Update() error {
	for id, player := range g.players { // Iterate through all players
		vx, vy := 0.0, 0.0

		// Only update local player's movement
		if id == g.localPlayerID {
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
		}

		// Update position
		player.x += vx
		player.y += vy

		// Prevent movement off-screen
		if player.x < 0 {
			player.x = 0
		}
		if player.x > ScreenWidth-PlayerSize {
			player.x = ScreenWidth - PlayerSize
		}
		if player.y < 0 {
			player.y = 0
		}
		if player.y > ScreenHeight-PlayerSize {
			player.y = ScreenHeight - PlayerSize
		}

		// Update direction if moving
		if vx != 0 || vy != 0 {
			player.angle = math.Atan2(vy, vx)
		}

		g.players[id] = player // Save updated player state
	}

	// // Shooting Mechanism
	// if g.player.cooldown > 0 {
	// 	g.player.cooldown--
	// }
	// if ebiten.IsKeyPressed(ebiten.KeySpace) && g.player.cooldown == 0 {
	// 	g.shootBullet()
	// 	g.player.cooldown = ShotCooldown
	// }


	// Bullet update logic
	for i := range g.bullets {
		if g.bullets[i].active {
			g.bullets[i].x += g.bullets[i].vx
			g.bullets[i].y += g.bullets[i].vy

			// Bullet out of bounds check
			if g.bullets[i].x < 0 || g.bullets[i].x > ScreenWidth || g.bullets[i].y < 0 || g.bullets[i].y > ScreenHeight {
				g.bullets[i].active = false
				continue
			}

			// Bullet collision with other players
			for pid, target := range g.players {
				if pid != g.bullets[i].ownerID && checkCollision(g.bullets[i], *target) {
					g.bullets[i].active = false
					target.health -= DamageAmount // Apply damage
					g.players[pid] = target // Update player health

					// Elimination check
					if target.health <= 0 {
						delete(g.players, pid) // Remove eliminated player
						fmt.Println("Player", pid, "eliminated!")
					}
				}
			}
		}
	}

	return nil
}

// 🚀 **Bullet Collision Check**
func checkCollision(b Bullet, p Player) bool {
	return b.x > p.x && b.x < p.x+PlayerSize && b.y > p.y && b.y < p.y+PlayerSize
}

// // Shoot a bullet
// func (g *Game) shootBullet() {
// 	vx := BulletSpeed * math.Cos(g.player.angle)
// 	vy := BulletSpeed * math.Sin(g.player.angle)
// 	g.bullets = append(g.bullets, Bullet{
// 		x:  g.player.x + PlayerSize/2,
// 		y:  g.player.y + PlayerSize/2,
// 		vx: vx, vy: vy,
// 		active: true,
// 		owner:  &g.player, // Tracks the shooter
// 	})
// }

// Draw renders everything
func (g *Game) Draw(screen *ebiten.Image) {
	for _, player := range g.players { // Draw all players
		// Draw player as a white rectangle
		ebitenutil.DrawRect(screen, player.x, player.y, PlayerSize, PlayerSize, color.White)

		// Draw facing direction
		centerX := player.x + PlayerSize/2
		centerY := player.y + PlayerSize/2
		endX := centerX + math.Cos(player.angle)*LineLength
		endY := centerY + math.Sin(player.angle)*LineLength
		ebitenutil.DrawLine(screen, centerX, centerY, endX, endY, color.RGBA{255, 0, 0, 255})

		// Draw health bar for each player
		g.drawHealthBar(screen, player)
	}

	// Draw bullets
	for _, b := range g.bullets {
		if b.active {
			ebitenutil.DrawRect(screen, b.x, b.y, BulletSize, BulletSize, color.RGBA{255, 255, 0, 255})
		}
	}
}

// 🚀 **Draw Health Bar Above Players**
func (g *Game) drawHealthBar(screen *ebiten.Image, player *Player) {
	barX := player.x - (HealthBarWidth-PlayerSize)/2
	barY := player.y - 5 // Position above player

	// Calculate health percentage
	healthPercentage := float64(player.health) / float64(MaxHealth)
	barCurrentWidth := HealthBarWidth * healthPercentage

	// Change health bar color based on health
	var healthColor color.Color
	if healthPercentage > 0.2 {
		healthColor = color.RGBA{0, 255, 0, 255} // Green when health is normal
	} else {
		healthColor = color.RGBA{255, 0, 0, 255} // Red when critically low
	}

	// Draw background (gray)
	ebitenutil.DrawRect(screen, barX, barY, HealthBarWidth, HealthBarHeight, color.RGBA{100, 100, 100, 255})

	// Draw health bar (green or red)
	ebitenutil.DrawRect(screen, barX, barY, barCurrentWidth, HealthBarHeight, healthColor)
}


// Layout defines the screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

// 🚀 **Generate a Unique Spawn Location**
func getRandomSpawn(existingPlayers map[string]*Player) (float64, float64) {
	rand.Seed(time.Now().UnixNano()) // Seed randomness

	for {
		x := rand.Float64()*(ScreenWidth-PlayerSize) + PlayerSize/2
		y := rand.Float64()*(ScreenHeight-PlayerSize) + PlayerSize/2

		// Ensure new spawn is not too close to an existing player
		overlapping := false
		for _, p := range existingPlayers {
			dist := (p.x-x)*(p.x-x) + (p.y-y)*(p.y-y)
			if dist < (PlayerSize * PlayerSize) {
				overlapping = true
				break
			}
		}

		if !overlapping {
			return x, y
		}
	}
}

func main() {
	game := &Game{
		players: make(map[string]*Player), // Initialize player map
	}

	// Assign a unique local player ID
	localID := "player_1" // In multiplayer, this would be dynamically assigned
	game.localPlayerID = localID

	// Get a random spawn position
	spawnX, spawnY := getRandomSpawn(game.players)

	// Create the local player with a unique ID and random spawn position
	game.players[localID] = &Player{
		id:     localID,
		x:      spawnX,
		y:      spawnY,
		health: MaxHealth,
	}

	// Set window properties
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("2D Battle Royale")

	// Run game loop
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}