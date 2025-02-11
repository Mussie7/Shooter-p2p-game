package peer

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"strings"
	"os/signal"
	"syscall"
	"time"

	"shooter/game"
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

var (
	DiscoveryServer = "192.168.0.100:5000" // Replace with actual local IP
	ActiveConnections = make(map[string]net.Conn) // Track connected peers
	Mutex            = &sync.Mutex{}
	SelfAddr         string // Store this peer's address
	GameInstance *game.Game // Reference to game instance (main.go)

)

//  Register this peer with the discovery server
func RegisterWithDiscovery(addr string) {
	conn, err := net.Dial("tcp", DiscoveryServer)
	if err != nil {
		fmt.Println("Error connecting to discovery server:", err)
		return
	}
	defer conn.Close()

	req := Request{Type: "register", Addr: addr}
	json.NewEncoder(conn).Encode(req)

	fmt.Println("Registered with discovery server as:", addr)
}

// **Send deregistration request when exiting**
func deregisterFromDiscovery() {
	conn, err := net.Dial("tcp", DiscoveryServer)
	if err != nil {
		fmt.Println("Error connecting to discovery server:", err)
		return
	}
	defer conn.Close()

	req := Request{Type: "deregister", Addr: SelfAddr}
	json.NewEncoder(conn).Encode(req)

	fmt.Println("Deregistered from discovery server:", SelfAddr)
}

// Handle SIGINT (CTRL+C) to clean up before exit
func HandleExit() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("Shutting down...")

		// Notify the discovery server
		deregisterFromDiscovery()

		// Close all active connections
		Mutex.Lock()
		for _, conn := range ActiveConnections {
			conn.Close()
		}
		ActiveConnections = make(map[string]net.Conn) // Clear connections
		Mutex.Unlock()

		os.Exit(0)
	}()
}

//  Get the list of peers from the discovery server
func GetPeers() []string {
    for retries := 0; retries < 3; retries++ {
        conn, err := net.Dial("tcp", DiscoveryServer)
        if err != nil {
            fmt.Println("Error connecting to discovery server (retrying)...", err)
            time.Sleep(2 * time.Second)
            continue
        }
        defer conn.Close()

        req := Request{Type: "get_peers"}
        json.NewEncoder(conn).Encode(req)

        var res Response
        json.NewDecoder(conn).Decode(&res)

        // Filter out self address
		filteredPeers := []string{}
        for _, peer := range res.Peers {
            if peer != SelfAddr {
                filteredPeers = append(filteredPeers, peer)
            }
        }

        return filteredPeers // Success case
    }
    return nil // Return empty after 3 failed attempts
}

//  Start TCP server to listen for peer connections
func StartPeerServer(addr string) {
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

		Mutex.Lock()
		if _, exists := ActiveConnections[peerAddr]; exists {
			Mutex.Unlock()
			conn.Close() // Close duplicate connection
			continue
		}
		ActiveConnections[peerAddr] = conn
		Mutex.Unlock()

		fmt.Println("Accepted connection from:", peerAddr)

		go handlePeerCommunication(conn)
	}
}

//  Connect to a discovered peer
func ConnectToPeer(peerAddr string) {
	Mutex.Lock()
	if _, exists := ActiveConnections[peerAddr]; exists {
		Mutex.Unlock()
		return //  Prevent duplicate connections
	}
	Mutex.Unlock()

	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		fmt.Println("Error connecting to peer:", peerAddr, err)
		return
	}

	Mutex.Lock()
	ActiveConnections[peerAddr] = conn
	Mutex.Unlock()

	fmt.Println("Connected to peer:", peerAddr)

	go handlePeerCommunication(conn)
}

func handlePeerCommunication(conn net.Conn) {
    defer conn.Close()

	peerAddr := conn.RemoteAddr().String()
    buffer := make([]byte, 1024)
    for {
        n, err := conn.Read(buffer)
        if err != nil {
            fmt.Println("Peer disconnected:", peerAddr)

			// Remove the peer from active connections
            Mutex.Lock()
            delete(ActiveConnections, peerAddr)
            Mutex.Unlock()

			// Notify the game to remove the player
			if GameInstance != nil {
				GameInstance.RemovePlayerAfterDelay(peerAddr)
			}
            return
        }

        var message map[string]interface{}
        err = json.Unmarshal(buffer[:n], &message)
        if err != nil {
            fmt.Println("Error decoding message:", err)
            continue
        }

        messageType, ok := message["type"].(string)
        if !ok {
            continue
        }

        // Handle movement updates
        if messageType == "move" {
            var moveMsg game.MovementMessage
            json.Unmarshal(buffer[:n], &moveMsg)
            if GameInstance != nil {
                GameInstance.UpdatePlayerPosition(moveMsg)
            }
        }

        // Handle shooting updates
        if messageType == "bullet" {
            var bulletMsg game.BulletMessage
            json.Unmarshal(buffer[:n], &bulletMsg)
            if GameInstance != nil {
                GameInstance.AddBulletFromPeer(bulletMsg)
            }
        }
    }
}

func SendUpdate(data interface{}) {
    jsonData, err := json.Marshal(data)
    if err != nil {
        fmt.Println("Error encoding update:", err)
        return
    }

    Mutex.Lock()
    defer Mutex.Unlock()
    for _, conn := range ActiveConnections {
        _, err := conn.Write(jsonData)
        if err != nil {
            fmt.Println("Error sending update:", err)
        }
    }
}