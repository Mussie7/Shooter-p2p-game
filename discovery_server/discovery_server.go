package main

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

var (
	peers = make(map[string]bool) // Store active peers
	mutex = &sync.Mutex{}
)

// Request structure from peers
type Request struct {
	Type string `json:"type"`
	Addr string `json:"addr,omitempty"`
}

// Response structure to peers
type Response struct {
	Peers []string `json:"peers"`
}

func main() {
	listener, err := net.Listen("tcp", ":5000") // Listen on port 5000
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Discovery Server is running on port 5000...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	var req Request
	if err := decoder.Decode(&req); err != nil {
		fmt.Println("Invalid request:", err)
		return
	}

	mutex.Lock()
	switch req.Type {
	case "register":
		peers[req.Addr] = true
		fmt.Println("Registered peer:", req.Addr)

	case "get_peers":
		var peerList []string
		for addr := range peers {
			peerList = append(peerList, addr)
		}
		response := Response{Peers: peerList}
		mutex.Unlock()

		encoder := json.NewEncoder(conn)
		encoder.Encode(response)
		return
	}
	mutex.Unlock()
}