package main

import (
	"os"
	"fmt"

	"shooter/game"
	"shooter/peer"
)

func main() {
	if len(os.Args) < 2 {
        fmt.Println("Usage: go run main.go <port>")
        os.Exit(1)
    }

    port := os.Args[1]  // Take port from CLI arguments

    // **Link game instance with peer networking**
    gameInstance := &game.Game{
        Players: make(map[string]*game.Player),
        ActiveConnections: peer.ActiveConnections,
		SendUpdate: peer.SendMovementUpdate, // Inject function

    }
	peer.GameInstance = gameInstance
    
	go peer.StartPeerServer("192.168.0.101:" + port) // Dynamic port
    gameInstance.MainGame(gameInstance)
}