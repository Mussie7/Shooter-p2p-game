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

func TestDeregisterFromDiscovery(t *testing.T) {
	mockDiscovery := startMockDiscoveryServer(t)
	defer mockDiscovery.Close()

	peer.DiscoveryServer = mockDiscovery.Listener.Addr().String()
	peer.SelfAddr = "192.168.0.100:8080"

	// Register the peer
	peer.RegisterWithDiscovery(peer.SelfAddr)

	time.Sleep(1 * time.Second) // Allow time for registration

	// Deregister the peer
	peer.DeregisterFromDiscovery()

	time.Sleep(1 * time.Second) // Allow time for deregistration

	// Check if the peer was removed
	peers := peer.GetPeers()
	for _, p := range peers {
		if p == peer.SelfAddr {
			t.Errorf("Deregistered peer %s still found in registered peers", peer.SelfAddr)
		}
	}
}

func TestMultiplePeersConnection(t *testing.T) {
	mockServer1 := startMockPeerServer(t)
	mockServer2 := startMockPeerServer(t)
	defer mockServer1.Close()
	defer mockServer2.Close()

	peerAddr1 := mockServer1.Listener.Addr().String()
	peerAddr2 := mockServer2.Listener.Addr().String()

	go peer.ConnectToPeer(peerAddr1)
	go peer.ConnectToPeer(peerAddr2)

	time.Sleep(2 * time.Second) // Allow connections

	peer.Mutex.Lock()
	_, exists1 := peer.ActiveConnections[peerAddr1]
	_, exists2 := peer.ActiveConnections[peerAddr2]
	peer.Mutex.Unlock()

	if !exists1 || !exists2 {
		t.Errorf("One or more peers failed to connect: %v %v", exists1, exists2)
	}
}

func TestBroadcastUpdateToAllPeers(t *testing.T) {
	mockServer1 := startMockPeerServer(t)
	mockServer2 := startMockPeerServer(t)
	defer mockServer1.Close()
	defer mockServer2.Close()

	peerAddr1 := mockServer1.Listener.Addr().String()
	peerAddr2 := mockServer2.Listener.Addr().String()

	go peer.ConnectToPeer(peerAddr1)
	go peer.ConnectToPeer(peerAddr2)

	time.Sleep(2 * time.Second) // Allow connections

	message := map[string]interface{}{
		"type": "move",
		"id":   "test_player",
		"x":    50,
		"y":    50,
		"angle": 90,
	}

	peer.SendUpdate(message)

	received1 := <-mockServer1.ReceivedMessages
	received2 := <-mockServer2.ReceivedMessages

	if received1["type"] != "move" || received2["type"] != "move" {
		t.Errorf("One or more peers did not receive the movement update")
	}
}

func TestHighLoadMultiplePeers(t *testing.T) {
	const numPeers = 10
	mockServers := make([]*mockServer, numPeers)
	peerAddresses := make([]string, numPeers)

	// Start multiple mock peer servers
	for i := 0; i < numPeers; i++ {
		mockServers[i] = startMockPeerServer(t)
		defer mockServers[i].Close()
		peerAddresses[i] = mockServers[i].Listener.Addr().String()
	}

	// Connect to all peers
	for _, addr := range peerAddresses {
		go peer.ConnectToPeer(addr)
	}

	time.Sleep(3 * time.Second) // Allow connections to stabilize

	peer.Mutex.Lock()
	for _, addr := range peerAddresses {
		if _, exists := peer.ActiveConnections[addr]; !exists {
			t.Errorf("Peer %s did not connect properly", addr)
		}
	}
	peer.Mutex.Unlock()
}

func TestEliminatedPlayersNotReceivingUpdates(t *testing.T) {
	mockServer := startMockPeerServer(t)
	defer mockServer.Close()

	peerAddr := mockServer.Listener.Addr().String()
	go peer.ConnectToPeer(peerAddr)

	time.Sleep(1 * time.Second) // Allow connection

	// Simulate player elimination
	message := map[string]interface{}{
		"type":  "eliminate",
		"id":    "test_player",
	}
	peer.SendUpdate(message)

	time.Sleep(1 * time.Second) // Allow time for processing

	// Send movement update
	moveMessage := map[string]interface{}{
		"type": "move",
		"id":   "test_player",
		"x":    100,
		"y":    100,
	}
	peer.SendUpdate(moveMessage)

	// Ensure the eliminated player did not receive the movement update
	select {
	case received := <-mockServer.ReceivedMessages:
		if received["type"] == "move" {
			t.Errorf("Eliminated player received movement update")
		}
	default:
		// No movement update received, test passes
	}
}

// ** Mock Peer Server**
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

func startMockDiscoveryServer(t *testing.T) *mockServer {
	server := newMockServer(t)
	registeredPeers := make(map[string]bool) // Stores peer registrations

	go server.ListenForRequests(func(conn net.Conn) {
		decoder := json.NewDecoder(conn)
		var req peer.Request
		if err := decoder.Decode(&req); err != nil {
			t.Errorf("Error decoding request: %v", err)
			return
		}

		peer.Mutex.Lock()
		if req.Type == "register" {
			registeredPeers[req.Addr] = true
		} else if req.Type == "get_peers" {
			var peerList []string
			for addr := range registeredPeers {
				peerList = append(peerList, addr)
			}
			resp := peer.Response{Peers: peerList}
			peer.Mutex.Unlock()
			json.NewEncoder(conn).Encode(resp)
			return
		}
		peer.Mutex.Unlock()
	})
	return server
}

// ** Mock Server Helper**
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