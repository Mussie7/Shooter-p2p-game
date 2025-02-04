package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

// Request structure for discovery server
type Request struct {
	Type string `json:"type"`
	Addr string `json:"addr,omitempty"`
}

// Response structure from discovery server
type Response struct {
	Peers []string `json:"peers"`
}

// const discoveryServer = "localhost:5000" // Change this if hosted remotely
const discoveryServer = "192.168.0.101:5000" // Replace with actual local IP

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run peer.go <port>")
		os.Exit(1)
	}

	port := os.Args[1]
	addr := fmt.Sprintf("localhost:%s", port)

	// Step 1: Register with Discovery Server
	registerWithDiscovery(addr)

	// Step 2: Fetch Peers List
	peerList := getPeers()
	fmt.Println("Discovered Peers:", peerList)

	// 🛠 Keep Running (Fix Deadlock)
	for {
		time.Sleep(time.Second) // Keeps the program alive
	}
}

// 📡 Register this peer with the discovery server
func registerWithDiscovery(addr string) {
	conn, err := net.Dial("tcp", discoveryServer)
	if err != nil {
		fmt.Println("Error connecting to discovery server:", err)
		return
	}
	defer conn.Close()

	req := Request{Type: "register", Addr: addr}
	json.NewEncoder(conn).Encode(req)

	fmt.Println("Registered with discovery server as:", addr)
}

// 🔎 Get the list of peers from the discovery server
func getPeers() []string {
	conn, err := net.Dial("tcp", discoveryServer)
	if err != nil {
		fmt.Println("Error connecting to discovery server:", err)
		return nil
	}
	defer conn.Close()

	req := Request{Type: "get_peers"}
	json.NewEncoder(conn).Encode(req)

	var res Response
	json.NewDecoder(conn).Decode(&res)
	return res.Peers
}