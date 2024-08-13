package tests

import (
	Miner "blockchain/miner"
	Tracker "blockchain/tracker"
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestMinerDiscovery checks the functionality of the tracker's miner registration and discovery mechanisms.
// The test ensures that:
// 1. Miners can successfully register with the tracker and appear in the discovery list.
// 2. The tracker correctly handles miner timeouts and removes miners that do not renew their registration.
// 3. The tracker correctly updates the list of active miners when new miners register and old ones time out.
// This function simulates the registration of multiple miners, checks the list of registered miners,
// waits for a timeout, adds another miner, and verifies the updated list to ensure it reflects the expected state.
func TestMinerDiscovery(t *testing.T) {
	tracker := Tracker.NewTracker(8080)
	tracker.Start()
	time.Sleep(1000 * time.Millisecond)

	miners := make([]*Miner.Miner, 0)
	for i := 0; i < 2; i++ {
		miner := Miner.NewMiner(3000+i, 8080)
		miner.Start()
		miners = append(miners, miner)
	}
	time.Sleep(500 * time.Millisecond)
	// initialize a mock miner at 3002
	request := Tracker.PortJson{Port: 3002}
	reqBytes, _ := json.Marshal(request)
	url := "http://localhost:8080/register"
	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		t.Fatalf("failed to connect to tracker")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to register to tracker")
	}
	var response Tracker.PortsJson
	_ = json.NewDecoder(resp.Body).Decode(&response)
	peers := response.Ports
	// should have 3 peers (including the mock miner)
	if len(peers) != 3 {
		t.Fatalf("wrong number of peers: %d\n", len(peers))
	}

	// wait for 3002 miner to timeout
	time.Sleep(1000 * time.Millisecond)
	// initialize a mock miner at 3003
	request = Tracker.PortJson{Port: 3003}
	reqBytes, _ = json.Marshal(request)
	resp, err = http.Post(url, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		t.Fatalf("failed to connect to tracker")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to register to tracker")
	}
	_ = json.NewDecoder(resp.Body).Decode(&response)
	peers = response.Ports
	// should still have 10 peers (including the mock miner)
	if len(peers) != 3 {
		t.Fatalf("wrong number of peers: %d\n", len(peers))
	}
	// 3009 should not be in peers
	for _, peer := range peers {
		if peer == 3002 {
			t.Fatalf("3002 does not time out")
		}
	}
	// cleanup everything
	for _, miner := range miners {
		miner.Shutdown()
	}
	tracker.Shutdown()
}
