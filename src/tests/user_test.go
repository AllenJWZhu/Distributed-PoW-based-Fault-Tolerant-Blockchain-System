package tests

import (
	"blockchain/user"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewUser tests the creation of a new user and verifies that attempting to retrieve miners without a running tracker results in an error.
// The test ensures that the NewUser function behaves correctly when it cannot communicate with a tracker,
// which is simulated by not starting a tracker server and expecting an error when calling GetRandomMiners.
func TestNewUser(t *testing.T) {
	trackerPort := 8000
	newUser := user.NewUser(trackerPort)

	// We cannot access the unexported fields directly
	// Instead, we can test the behavior of the newUser through its methods
	// For example, we can call GetRandomMiners and check if it returns an error
	_, err := newUser.GetRandomMiners()
	if err == nil {
		t.Error("Expected an error when calling GetRandomMiners with no running tracker, but got nil")
	}
}

// TestGetRandomMiners tests the retrieval of random miners from the tracker.
// This test sets up a mock tracker server to respond with a list of miners and checks if the user can correctly retrieve and parse this list.
// It verifies the integration of the user client with the tracker server API, ensuring the user client handles responses correctly.
func TestGetRandomMiners(t *testing.T) {
	miners := []int{8001, 8002, 8003, 8004, 8005, 8006, 8007, 8008, 8009, 8010}
	mockTracker := newMockTracker(miners)

	// Setup a mock tracker server to handle miner retrieval requests
	trackerServer := httptest.NewServer(http.HandlerFunc(mockTracker.handleGetMiners))
	defer trackerServer.Close()

	// Create a new user client configured to use the mock tracker
	newUser := user.NewUser(extractPort(trackerServer.URL))

	// Retrieve random miners from the mock tracker
	randomMiners, err := newUser.GetRandomMiners()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check if the number of miners retrieved matches the expected count
	if len(randomMiners) != N {
		t.Errorf("Expected %d random miners, but got %d", N, len(randomMiners))
	}
}

// extractPort extracts the port number from a URL.
// This utility function parses out the port number from a given URL string, which is useful for setting up clients that need to connect to a server running on a dynamic port.
// The function assumes the URL starts directly with the hostname or IP address.
func extractPort(url string) int {
	_, portStr, _ := net.SplitHostPort(url[7:])
	var port int
	_, err := fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return 0
	}
	return port
}
