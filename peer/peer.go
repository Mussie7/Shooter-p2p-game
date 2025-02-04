package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"strings"
	// "time"
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

const discoveryServer = "192.168.0.101:5000" // Replace with actual local IP

var (
	activeConnections = make(map[string]net.Conn) // Track connected peers
	mutex            = &sync.Mutex{}
	selfAddr         string // Store this peer's address
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run peer.go <port>")
		os.Exit(1)
	}

	port := os.Args[1]
	selfAddr = fmt.Sprintf("192.168.0.101:%s", port) // Store self address

	// Step 1: Register with Discovery Server
	registerWithDiscovery(selfAddr)

	// Step 2: Start Listening for TCP Connections
	go startPeerServer(selfAddr)

	// Step 3: Fetch and Connect to Peers
	peerList := getPeers()
	fmt.Println("Discovered Peers:", peerList)

	for _, peer := range peerList {
		if peer != selfAddr { // ✅ Prevent self-connections
			go connectToPeer(peer)
		}
	}

	// Keep the program running
	select {}
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

	// 🔥 Filter out self-address before returning
	filteredPeers := []string{}
	for _, peer := range res.Peers {
		if peer != selfAddr { // ✅ Prevent listing self
			filteredPeers = append(filteredPeers, peer)
		}
	}

	return filteredPeers
}


// 🚀 Start TCP server to listen for peer connections
func startPeerServer(addr string) {
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", strings.Split(addr, ":")[1]))
	if err != nil {
		fmt.Println("Error starting peer server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Listening for peer connections on", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting peer connection:", err)
			continue
		}

		peerAddr := conn.RemoteAddr().String()

		mutex.Lock()
		if _, exists := activeConnections[peerAddr]; exists {
			mutex.Unlock()
			conn.Close() // Close duplicate connection
			continue
		}
		activeConnections[peerAddr] = conn
		mutex.Unlock()

		fmt.Println("Accepted connection from:", peerAddr)

		go handlePeerCommunication(conn)
	}
}

// 🔗 Connect to a discovered peer
func connectToPeer(peerAddr string) {
	mutex.Lock()
	if _, exists := activeConnections[peerAddr]; exists {
		mutex.Unlock()
		return // ✅ Prevent duplicate connections
	}
	mutex.Unlock()

	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		fmt.Println("Error connecting to peer:", peerAddr, err)
		return
	}

	mutex.Lock()
	activeConnections[peerAddr] = conn
	mutex.Unlock()

	fmt.Println("Connected to peer:", peerAddr)

	go handlePeerCommunication(conn)
}

// 📩 Handle messages from peers
func handlePeerCommunication(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Peer disconnected:", conn.RemoteAddr().String())
			mutex.Lock()
			delete(activeConnections, conn.RemoteAddr().String())
			mutex.Unlock()
			return
		}
		message := string(buffer[:n])
		fmt.Println("Received from", conn.RemoteAddr().String(), ":", message)
	}
}
