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

const discoveryServer = "192.168.0.101:5000" // Replace with actual local IP

var (
	ActiveConnections = make(map[string]net.Conn) // Track connected peers
	mutex            = &sync.Mutex{}
	selfAddr         string // Store this peer's address
	GameInstance *game.Game // Reference to game instance (main.go)

)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run peer.go <port>")
		os.Exit(1)
	}

	port := os.Args[1]
	selfAddr = fmt.Sprintf("192.168.0.101:%s", port) // Store self address

	// Handle graceful shutdown
	handleExit()

	// Step 1: Register with Discovery Server
	registerWithDiscovery(selfAddr)

	// Step 2: Start Listening for TCP Connections
	go StartPeerServer(selfAddr)

	// Step 3: Fetch and Connect to Peers
	peerList := getPeers()
	fmt.Println("Discovered Peers:", peerList)

	for _, peer := range peerList {
		if peer != selfAddr { //  Prevent self-connections
			go connectToPeer(peer)
		}
	}

	// Keep the program running
	select {}
}

//  Register this peer with the discovery server
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

// **Send deregistration request when exiting**
func deregisterFromDiscovery() {
	conn, err := net.Dial("tcp", discoveryServer)
	if err != nil {
		fmt.Println("Error connecting to discovery server:", err)
		return
	}
	defer conn.Close()

	req := Request{Type: "deregister", Addr: selfAddr}
	json.NewEncoder(conn).Encode(req)

	fmt.Println("Deregistered from discovery server:", selfAddr)
}

// Handle SIGINT (CTRL+C) to clean up before exit
func handleExit() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		deregisterFromDiscovery() // Deregister before exiting
		os.Exit(0)
	}()
}

//  Get the list of peers from the discovery server
func getPeers() []string {
    for retries := 0; retries < 3; retries++ {
        conn, err := net.Dial("tcp", discoveryServer)
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

        filteredPeers := []string{}
        for _, peer := range res.Peers {
            if peer != selfAddr {
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

		mutex.Lock()
		if _, exists := ActiveConnections[peerAddr]; exists {
			mutex.Unlock()
			conn.Close() // Close duplicate connection
			continue
		}
		ActiveConnections[peerAddr] = conn
		mutex.Unlock()

		fmt.Println("Accepted connection from:", peerAddr)

		go handlePeerCommunication(conn)
	}
}

//  Connect to a discovered peer
func connectToPeer(peerAddr string) {
	mutex.Lock()
	if _, exists := ActiveConnections[peerAddr]; exists {
		mutex.Unlock()
		return //  Prevent duplicate connections
	}
	mutex.Unlock()

	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		fmt.Println("Error connecting to peer:", peerAddr, err)
		return
	}

	mutex.Lock()
	ActiveConnections[peerAddr] = conn
	mutex.Unlock()

	fmt.Println("Connected to peer:", peerAddr)

	go handlePeerCommunication(conn)
}

func handlePeerCommunication(conn net.Conn) {
    defer conn.Close()

    buffer := make([]byte, 1024)
    for {
        n, err := conn.Read(buffer)
        if err != nil {
            fmt.Println("Peer disconnected:", conn.RemoteAddr().String())
            mutex.Lock()
            delete(ActiveConnections, conn.RemoteAddr().String())
            mutex.Unlock()
            return
        }

        var message game.MovementMessage
        err = json.Unmarshal(buffer[:n], &message)
        if err != nil {
            fmt.Println("Error decoding message:", err)
            continue
        }

        // **Update game state**
        if message.Type == "move" && GameInstance != nil {
            GameInstance.UpdatePlayerPosition(message)
        }
    }
}

func SendMovementUpdate(msg game.MovementMessage) {
    data, err := json.Marshal(msg)
    if err != nil {
        fmt.Println("Error encoding movement update:", err)
        return
    }

    mutex.Lock()
    defer mutex.Unlock()
    for _, conn := range ActiveConnections {
        _, err := conn.Write(data)
        if err != nil {
            fmt.Println("Error sending movement update:", err)
        }
    }
}