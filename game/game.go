package game

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"net"
	"sync"
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

var mutex sync.Mutex

// Player struct (now includes velocity for smoother movement updates)
type Player struct {
	id       string  // Unique player ID
	x, y     float64 // Position
	angle    float64 // Facing direction
	health   int     // Health bar
	cooldown int     // Shooting cooldown
	eliminated bool    // New: Marks player as eliminated

}

// MovementMessage struct (sent to peers when a player moves)
type MovementMessage struct {
	Type  string  `json:"type"`  // "move"
	ID    string  `json:"id"`    // Player ID
	X     float64 `json:"x"`     // Updated X position
	Y     float64 `json:"y"`     // Updated Y position
	Angle float64 `json:"angle"` // Direction the player is facing
}

// BulletMessage struct (sent to peers when a bullet is fired)
type BulletMessage struct {
	Type    string  `json:"type"`  // "bullet"
	OwnerID string  `json:"owner_id"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	VX      float64 `json:"vx"`
	VY      float64 `json:"vy"`
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
	Players       map[string]*Player // Stores all players
	bullets       []Bullet           // Stores all bullets
	LocalPlayerID string             // ID of the local player
	ActiveConnections map[string]net.Conn  // Stores active TCP connections to peers
	SendUpdate func(interface{}) // Field for sending updates

}

func (g *Game) Update() error {
	player, exists := g.Players[g.LocalPlayerID]
    if !exists || player.eliminated { // Prevent updates for eliminated players
        return nil
    }

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

	// Only send updates if movement occurred
	if vx != 0 || vy != 0 {
		player.angle = math.Atan2(vy, vx)
		player.x += vx
		player.y += vy

		// Prevent moving off-screen
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

		// Send movement update to peers
		g.sendMovementUpdate(player)
	}

	// Shooting Mechanism
	if g.Players[g.LocalPlayerID].cooldown > 0 {
		g.Players[g.LocalPlayerID].cooldown--
	}
	if ebiten.IsKeyPressed(ebiten.KeySpace) && g.Players[g.LocalPlayerID].cooldown == 0 {
		g.shootBullet()
		g.Players[g.LocalPlayerID].cooldown = ShotCooldown
	}


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
			for pid, target := range g.Players {
				if pid != g.bullets[i].ownerID && checkCollision(g.bullets[i], target, g) {
					g.bullets[i].active = false
					target.health -= DamageAmount // Apply damage
					g.Players[pid] = target // Update player health

					// Elimination check
					if target.health <= 0 {
						delete(g.Players, pid) // Remove eliminated player
						fmt.Println("Player", pid, "eliminated!")
					}
				}
			}
		}
	}

	return nil
}

func (g *Game) sendMovementUpdate(player *Player) {
	message := MovementMessage{
		Type:  "move",
		ID:    player.id,
		X:     player.x,
		Y:     player.y,
		Angle: player.angle,
	}

	// Call the injected function
	if g.SendUpdate != nil {
		g.SendUpdate(message)
	}
}

func (g *Game) UpdatePlayerPosition(msg MovementMessage) {
    mutex.Lock()
    defer mutex.Unlock()

    if player, exists := g.Players[msg.ID]; exists {
        player.x = msg.X
        player.y = msg.Y
        player.angle = msg.Angle
    } else {
        // **Create new player if they don't exist**
        g.Players[msg.ID] = &Player{
            id:     msg.ID,
            x:      msg.X,
            y:      msg.Y,
            angle:  msg.Angle,
            health: MaxHealth,
        }
    }
}

// **Bullet Collision Check**
func checkCollision(b Bullet, p *Player, g *Game) bool {
	if b.x > p.x && b.x < p.x+PlayerSize && b.y > p.y && b.y < p.y+PlayerSize {
		p.health -= DamageAmount
		fmt.Println("Player", p.id, "hit! New health:", p.health)

		if p.health <= 0 && !p.eliminated {
			fmt.Println("Player", p.id, "eliminated!")
			p.eliminated = true // Mark as eliminated
			go g.RemovePlayerAfterDelay(p.id) // Remove after delay
		}
		return true
	}
	return false
}

func (g *Game) RemovePlayerAfterDelay(playerID string) {
	time.Sleep(3 * time.Second) // Wait 3 seconds before removal

	mutex.Lock()
	defer mutex.Unlock()

	if _, exists := g.Players[playerID]; exists {
		fmt.Println("Removing player:", playerID)
		delete(g.Players, playerID) // Now safe to remove
	}
}

// Shoot a bullet and send an update to peers
func (g *Game) shootBullet() {
	vx := BulletSpeed * math.Cos(g.Players[g.LocalPlayerID].angle)
	vy := BulletSpeed * math.Sin(g.Players[g.LocalPlayerID].angle)
	newBullet := Bullet{
		x:       g.Players[g.LocalPlayerID].x + PlayerSize/2,
		y:       g.Players[g.LocalPlayerID].y + PlayerSize/2,
		vx:      vx, 
		vy:      vy,
		active:  true,
		ownerID: g.LocalPlayerID, // Identify shooter
	}

	g.bullets = append(g.bullets, newBullet)

	// Send bullet data to all peers
	if g.SendUpdate != nil {
		bulletUpdate := BulletMessage{
			Type:    "bullet",
			OwnerID: newBullet.ownerID,
			X:       newBullet.x,
			Y:       newBullet.y,
			VX:      newBullet.vx,
			VY:      newBullet.vy,
		}
		g.SendUpdate(bulletUpdate)
	}
}

// Add a bullet from a received peer message
func (g *Game) AddBulletFromPeer(msg BulletMessage) {
	mutex.Lock()
	defer mutex.Unlock()

	newBullet := Bullet{
		x:       msg.X,
		y:       msg.Y,
		vx:      msg.VX,
		vy:      msg.VY,
		active:  true,
		ownerID: msg.OwnerID,
	}

	g.bullets = append(g.bullets, newBullet)
}

// Draw renders everything
func (g *Game) Draw(screen *ebiten.Image) {
	for _, player := range g.Players { // Draw all players
		if player.eliminated { // Skip eliminated players
			continue
		}

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

// **Draw Health Bar Above Players**
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

// **Generate a Unique Spawn Location**
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

func (g *Game) MainGame(game *Game) {
	// Get a random spawn position
	spawnX, spawnY := getRandomSpawn(game.Players)

	// Create the local player with a unique ID and random spawn position
	game.Players[game.LocalPlayerID] = &Player{
		id:     game.LocalPlayerID,
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