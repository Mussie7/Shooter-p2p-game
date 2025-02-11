package peer_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"shooter/peer"
)
// ** Test Peer Connection**
func TestPeerConnection(t *testing.T) {
	mockServer := startMockPeerServer(t)
	defer mockServer.Close()

	peerAddr := mockServer.Listener.Addr().String()
	go peer.ConnectToPeer(peerAddr)

	// Allow some time for connection
	time.Sleep(1 * time.Second)

	peer.Mutex.Lock()
	_, exists := peer.ActiveConnections[peerAddr]
	peer.Mutex.Unlock()

	if !exists {
		t.Errorf("Expected connection to peer %s, but it was not established", peerAddr)
	}
}

// ** Test Send and Receive Updates**
func TestSendAndReceiveUpdates(t *testing.T) {
	mockServer := startMockPeerServer(t)
	defer mockServer.Close()

	peerAddr := mockServer.Listener.Addr().String()
	go peer.ConnectToPeer(peerAddr)

	time.Sleep(1 * time.Second) // Allow time for connection

	message := map[string]interface{}{
		"type": "test_message",
		"data": "Hello, Peer!",
	}

	peer.SendUpdate(message)

	receivedMessage := <-mockServer.ReceivedMessages
	if receivedMessage["type"] != "test_message" {
		t.Errorf("Expected message type 'test_message', got %v", receivedMessage["type"])
	}
}

// **🛠️ Mock Peer Server**
func startMockPeerServer(t *testing.T) *mockServer {
	server := newMockServer(t)
	go server.ListenForRequests(func(conn net.Conn) {
		decoder := json.NewDecoder(conn)
		var message map[string]interface{}
		if err := decoder.Decode(&message); err != nil {
			t.Errorf("Error decoding peer message: %v", err)
			return
		}
		server.ReceivedMessages <- message
	})
	return server
}

// **🛠️ Mock Server Helper**
type mockServer struct {
	Listener        net.Listener
	ReceivedMessages chan map[string]interface{}
}

func newMockServer(t *testing.T) *mockServer {
	listener, err := net.Listen("tcp", "127.0.0.1:0") // Random available port
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}

	return &mockServer{
		Listener:        listener,
		ReceivedMessages: make(chan map[string]interface{}, 10),
	}
}

func (s *mockServer) ListenForRequests(handler func(conn net.Conn)) {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			break // Stop accepting when closed
		}
		go handler(conn)
	}
}

func (s *mockServer) Close() {
	s.Listener.Close()
}