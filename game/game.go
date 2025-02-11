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
	HealthBarWidth  = 35  // New: Health bar width
	HealthBarHeight = 3   // New: Health bar height
)

var (
	mutex sync.Mutex
	tankImage *ebiten.Image

)

// Player struct (now includes velocity for smoother movement updates)
type Player struct {
	ID       string  // Unique player ID
	X, Y     float64 // Position
	Angle    float64 // Facing direction
	Health   int     // Health bar
	cooldown int     // Shooting cooldown
	eliminated bool    // New: Marks player as eliminated
	Image  *ebiten.Image // Store the player's tank sprite


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
	X, Y     float64
	vx, vy   float64
	Active   bool
	OwnerID  string // ID of the player who fired it
}

// Game struct (supports multiple players)
type Game struct {
	Players       map[string]*Player // Stores all players
	Bullets       []Bullet           // Stores all bullets
	LocalPlayerID string             // ID of the local player
	ActiveConnections map[string]net.Conn  // Stores active TCP connections to peers
	SendUpdate func(interface{}) // Field for sending updates

}

// LoadAssets loads the tank sprite
func LoadAssets() {
    img, _, err := ebitenutil.NewImageFromFile("assets/green_tank.png")
    if err != nil {
        log.Fatal("Failed to load tank sprite:", err)
    }
    tankImage = img
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
		player.Angle = math.Atan2(vy, vx)
		player.X += vx
		player.Y += vy

		// Prevent moving off-screen
		if player.X < 0 {
			player.X = 0
		}
		if player.X > ScreenWidth-PlayerSize {
			player.X = ScreenWidth - PlayerSize
		}
		if player.Y < 0 {
			player.Y = 0
		}
		if player.Y > ScreenHeight-PlayerSize {
			player.Y = ScreenHeight - PlayerSize
		}

		// Send movement update to peers
		g.sendMovementUpdate(player)
	}

	// Shooting Mechanism
	if g.Players[g.LocalPlayerID].cooldown > 0 {
		g.Players[g.LocalPlayerID].cooldown--
	}
	if ebiten.IsKeyPressed(ebiten.KeySpace) && g.Players[g.LocalPlayerID].cooldown == 0 {
		g.ShootBullet()
		g.Players[g.LocalPlayerID].cooldown = ShotCooldown
	}


	// Bullet update logic
	for i := range g.Bullets {
		if g.Bullets[i].Active {
			g.Bullets[i].X += g.Bullets[i].vx
			g.Bullets[i].Y += g.Bullets[i].vy

			// Bullet out of bounds check
			if g.Bullets[i].X < 0 || g.Bullets[i].X > ScreenWidth || g.Bullets[i].Y < 0 || g.Bullets[i].Y > ScreenHeight {
				g.Bullets[i].Active = false
				continue
			}

			// Bullet collision with other players
			for pid, target := range g.Players {
				if pid != g.Bullets[i].OwnerID && CheckCollision(g.Bullets[i], target, g) {
					g.Bullets[i].Active = false
					target.Health -= DamageAmount // Apply damage
					g.Players[pid] = target // Update player health

					// Elimination check
					if target.Health <= 0 {
						g.RemovePlayerAfterDelay(pid)
						// delete(g.Players, pid) // Remove eliminated player
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
		ID:    player.ID,
		X:     player.X,
		Y:     player.Y,
		Angle: player.Angle,
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
        player.X = msg.X
        player.Y = msg.Y
        player.Angle = msg.Angle
    } else {
        // **Create new player if they don't exist**
        g.Players[msg.ID] = &Player{
            ID:     msg.ID,
            X:      msg.X,
            Y:      msg.Y,
            Angle:  msg.Angle,
            Health: MaxHealth,
        }
    }
}

// **Bullet Collision Check**
func CheckCollision(b Bullet, p *Player, g *Game) bool {
	if b.X > p.X && b.X < p.X+PlayerSize && b.Y > p.Y && b.Y < p.Y+PlayerSize {
		p.Health -= DamageAmount
		fmt.Println("Player", p.ID, "hit! New health:", p.Health)

		if p.Health <= 0 && !p.eliminated {
			fmt.Println("Player", p.ID, "eliminated!")
			p.eliminated = true // Mark as eliminated
			go g.RemovePlayerAfterDelay(p.ID) // Remove after delay
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
func (g *Game) ShootBullet() {
	vx := BulletSpeed * math.Cos(g.Players[g.LocalPlayerID].Angle)
	vy := BulletSpeed * math.Sin(g.Players[g.LocalPlayerID].Angle)
	newBullet := Bullet{
		X:       g.Players[g.LocalPlayerID].X + PlayerSize/2,
		Y:       g.Players[g.LocalPlayerID].Y + PlayerSize/2,
		vx:      vx, 
		vy:      vy,
		Active:  true,
		OwnerID: g.LocalPlayerID, // Identify shooter
	}

	g.Bullets = append(g.Bullets, newBullet)

	// Send bullet data to all peers
	if g.SendUpdate != nil {
		bulletUpdate := BulletMessage{
			Type:    "bullet",
			OwnerID: newBullet.OwnerID,
			X:       newBullet.X,
			Y:       newBullet.Y,
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
		X:       msg.X,
		Y:       msg.Y,
		vx:      msg.VX,
		vy:      msg.VY,
		Active:  true,
		OwnerID: msg.OwnerID,
	}

	g.Bullets = append(g.Bullets, newBullet)
}

// Draw renders everything
func (g *Game) Draw(screen *ebiten.Image) {
	for _, player := range g.Players { // Draw all players

		if player.Image == nil {
            player.Image = tankImage // Assign the tank sprite to each player
        }

        op := &ebiten.DrawImageOptions{}
		scale := 0.15 // Adjust this value as needed
		op.GeoM.Scale(scale, scale) // Scale the sprite
        op.GeoM.Translate(-float64(player.Image.Bounds().Dx())*scale/2, -float64(player.Image.Bounds().Dy())*scale/2) // Center the rotation
        op.GeoM.Rotate(player.Angle) // Rotate the sprite
        op.GeoM.Translate(player.X, player.Y) // Position the sprite at the player's location

        screen.DrawImage(player.Image, op) // Render the tank sprite

		if player.eliminated { // Skip eliminated players
			continue
		}

		// Draw health bar for each player
		g.drawHealthBar(screen, player)
	}

	// Draw bullets
	for _, b := range g.Bullets {
		if b.Active {
			ebitenutil.DrawRect(screen, b.X - 12, b.Y - 12, BulletSize, BulletSize, color.RGBA{255, 255, 0, 255})
		}
	}
}

// **Draw Health Bar Above Players**
func (g *Game) drawHealthBar(screen *ebiten.Image, player *Player) {
	scale := 0.15 // The same scale used for the tank sprite
	tankHeight := float64(tankImage.Bounds().Dy()) * scale

    // Calculate health percentage
    healthPercentage := float64(player.Health) / float64(MaxHealth)
    barCurrentWidth := HealthBarWidth * healthPercentage

    // Calculate the position of the health bar based on the player's rotation
    barX := player.X - barCurrentWidth/2
    barY := player.Y - tankHeight/2 - 10 // Position above player (adjust as needed)

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
			dist := (p.X-x)*(p.X-x) + (p.Y-y)*(p.Y-y)
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
	// Load tank sprite
	LoadAssets()

	// Get a random spawn position
	spawnX, spawnY := getRandomSpawn(game.Players)

	// Create the local player with a unique ID and random spawn position
	game.Players[game.LocalPlayerID] = &Player{
		ID:     game.LocalPlayerID,
		X:      spawnX,
		Y:      spawnY,
		Health: MaxHealth,
	}

	// Set window properties
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("2D Battle Royale")

	// Run game loop
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}