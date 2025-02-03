package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var peers = make(map[string]bool) // Keep track of discovered peers
var mutex = &sync.Mutex{}
var broadcastAddr = "255.255.255.255:9999"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run peer.go <port>")
		os.Exit(1)
	}

	port := os.Args[1]

	go startUDPListener()      // Start listening for broadcasts
	go broadcastPresence(port) // Send "HELLO" to discover others

	select {} // Keep running
}

// 🚀 **Listen for UDP Broadcasts from Other Peers**
func startUDPListener() {
	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:9999")
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error starting UDP listener:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Listening for peer discovery on port 9999")

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading UDP broadcast:", err)
			continue
		}

		message := string(buf[:n])
		if strings.HasPrefix(message, "HELLO") {
			handlePeerDiscovery(message, remoteAddr)
		}
	}
}

// 🚀 **Process Discovered Peers**
func handlePeerDiscovery(message string, remoteAddr *net.UDPAddr) {
	peerPort := strings.Split(message, ":")[1]
	peerAddress := fmt.Sprintf("%s:%s", remoteAddr.IP.String(), peerPort)

	mutex.Lock()
	if _, exists := peers[peerAddress]; exists {
		mutex.Unlock()
		return // Ignore already discovered peers
	}
	peers[peerAddress] = true
	mutex.Unlock()

	fmt.Println("Discovered peer:", peerAddress)
}

// 🚀 **Broadcast Presence Until a Peer is Found**
func broadcastPresence(port string) {
	addr, err := net.ResolveUDPAddr("udp", broadcastAddr)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Error dialing UDP broadcast address:", err)
		return
	}
	defer conn.Close()

	message := fmt.Sprintf("HELLO:%s", port)
	for {
		// mutex.Lock()
		// if len(peers) > 0 {
		// 	mutex.Unlock()
		// 	break // Stop broadcasting once a peer is found
		// }
		// mutex.Unlock()

		_, err := conn.Write([]byte(message))
		if err != nil {
			fmt.Println("Error sending UDP broadcast:", err)
		} else {
			fmt.Println("Sent UDP broadcast:", message)
		}
		time.Sleep(3 * time.Second) // Retry every 3 seconds
	}
}
