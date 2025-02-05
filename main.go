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

    playerAddr := fmt.Sprintf("192.168.0.101:%s", port) // Update with actual LAN IP
	peer.SelfAddr = playerAddr // Store self address in peer package

	// Register player with the discovery server
	peer.RegisterWithDiscovery(playerAddr)

	// Create the game instance
    gameInstance := &game.Game{
        Players: make(map[string]*game.Player),
        ActiveConnections: peer.ActiveConnections,
		SendUpdate: peer.SendMovementUpdate, // Inject function

    }
	// Set game instance in peer package
	peer.GameInstance = gameInstance

	// Start TCP server to accept peer connections
	go peer.StartPeerServer(playerAddr)
    
	// Get discovered peers and connect to them
	peerList := peer.GetPeers()
	for _, peerAddr := range peerList {
		if peerAddr != playerAddr {
			go peer.ConnectToPeer(peerAddr)
		}
	}

	// Start the game
	gameInstance.MainGame(gameInstance)
}