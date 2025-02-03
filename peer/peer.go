package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

var peers = make(map[string]net.Conn)
var mutex = &sync.Mutex{}

type Message struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run peer.go <port> [peer addresses...]")
		os.Exit(1)
	}

	port := os.Args[1]
	initialPeers := os.Args[2:]

	go startServer(port)

	for _, address := range initialPeers {
		go connectToPeer(address)
	}

	go readUserInput()

	select {} // Keep the main function running
}

func startServer(port string) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("Listening on port", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		mutex.Lock()
		peers[conn.RemoteAddr().String()] = conn
		mutex.Unlock()

		fmt.Println("Connected to", conn.RemoteAddr().String())

		go handleConnection(conn)
	}
}

func connectToPeer(address string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Error connecting to peer:", err)
		return
	}

	mutex.Lock()
	peers[conn.RemoteAddr().String()] = conn
	mutex.Unlock()

	fmt.Println("Connected to peer", address)

	go handleConnection(conn)
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Disconnected from", conn.RemoteAddr().String())
			removePeer(conn)
			return
		}

		var msg Message
		err = json.Unmarshal([]byte(message), &msg)
		if err != nil {
			fmt.Println("Error decoding message:", err)
			continue
		}

		switch msg.Type {
		case "move":
			fmt.Println("Player moved:", msg.Data)
		case "chat":
			fmt.Println("Chat message:", msg.Data)
		// Handle other message types...
		}

		broadcastMessage(conn, message)
	}
}

func broadcastMessage(sender net.Conn, message string) {
	mutex.Lock()
	defer mutex.Unlock()

	for addr, peer := range peers {
		if peer != sender {
			_, err := peer.Write([]byte(message))
			if err != nil {
				fmt.Println("Error sending message to", addr)
			}
		}
	}
}

func removePeer(conn net.Conn) {
	mutex.Lock()
	defer mutex.Unlock()

	delete(peers, conn.RemoteAddr().String())
}

func sendMoveMessage(conn net.Conn, x, y int) {
	message := Message{
		Type: "move",
		Data: fmt.Sprintf("x:%d,y:%d", x, y),
	}
	sendMessage(conn, message)
}

func sendChatMessage(data string) {
	message := Message{
		Type: "chat",
		Data: data,
	}
	broadcastMessageToAll(message)
}

func sendMessage(conn net.Conn, message Message) {
	data, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Error encoding message:", err)
		return
	}
	conn.Write(data)
	conn.Write([]byte("\n"))
}

func broadcastMessageToAll(message Message) {
	mutex.Lock()
	defer mutex.Unlock()

	for _, peer := range peers {
		sendMessage(peer, message)
	}
}

func readUserInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter message: ")
		text, _ := reader.ReadString('\n')
		sendChatMessage(text)
	}
}