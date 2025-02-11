package game_test

import (
	"testing"

	"shooter/game" // Import the actual package
)

// ** Test Player Movement**
func TestPlayerMovement(t *testing.T) {
	gameInstance := &game.Game{
		Players: make(map[string]*game.Player),
	}

	// Create a test player
	playerID := "player1"
	gameInstance.Players[playerID] = &game.Player{ID: playerID, X: 100, Y: 100, Angle: 0}

	// Simulate movement
	moveMsg := game.MovementMessage{ID: playerID, X: 150, Y: 200, Angle: 1.57}
	gameInstance.UpdatePlayerPosition(moveMsg)

	// Verify the position update
	if gameInstance.Players[playerID].X != 150 || gameInstance.Players[playerID].Y != 200 || gameInstance.Players[playerID].Angle != 1.57 {
		t.Errorf("Expected position (150, 200) but got (%f, %f)", gameInstance.Players[playerID].X, gameInstance.Players[playerID].Y)
	}
}

// ** Test Bullet Creation**
func TestShootBullet(t *testing.T) {
	gameInstance := &game.Game{
		Players: make(map[string]*game.Player),
	}

	// Add a test player
	playerID := "player1"
	gameInstance.Players[playerID] = &game.Player{ID: playerID, X: 100, Y: 100, Angle: 0}

	// Fire a bullet
	gameInstance.LocalPlayerID = playerID
	gameInstance.ShootBullet()

	// Verify bullet was added
	if len(gameInstance.Bullets) != 1 {
		t.Errorf("Expected 1 bullet, got %d", len(gameInstance.Bullets))
	}

	// Verify bullet properties
	bullet := gameInstance.Bullets[0]
	if bullet.X != 100+game.PlayerSize/2 || bullet.Y != 100+game.PlayerSize/2 {
		t.Errorf("Bullet spawned at incorrect location (%f, %f)", bullet.X, bullet.Y)
	}
}

// ** Test Bullet Collision**
func TestBulletCollision(t *testing.T) {
	gameInstance := &game.Game{
		Players: make(map[string]*game.Player),
	}

	// Add a test player
	playerID := "player1"
	gameInstance.Players[playerID] = &game.Player{ID: playerID, X: 50, Y: 50, Health: 100}

	// Create a bullet
	bullet := game.Bullet{X: 55, Y: 55, Active: true, OwnerID: "player2"}

	// Check if collision occurs
	if !game.CheckCollision(bullet, gameInstance.Players[playerID], gameInstance) {
		t.Errorf("Expected collision but none occurred")
	}

	// Verify player health reduction
	if gameInstance.Players[playerID].Health != 95 {
		t.Errorf("Expected health 95, but got %d", gameInstance.Players[playerID].Health)
	}
}