package tests

import (
	"blockchain/blockchain"
	Miner "blockchain/miner"
	Tracker "blockchain/tracker"
	User "blockchain/user"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestMaliciousUser - Tests the behavior of miners when a malicious user tries to submit a tampered or duplicated post.
func TestMaliciousUser(t *testing.T) {
	tracker := Tracker.NewTracker(8082)
	tracker.Start()
	defer tracker.Shutdown()
	time.Sleep(1000 * time.Millisecond)

	// Create a legitimate miner
	legitimateMiner := Miner.NewMiner(3003, 8082)
	legitimateMiner.Start()
	defer legitimateMiner.Shutdown()

	// Create a malicious user
	maliciousUser := User.NewUser(8082)

	// Malicious user attempts to post a legitimate message
	legitimateContent := "Legitimate content"
	err := maliciousUser.WritePost(legitimateContent)
	if err != nil {
		t.Fatalf("error when posting legitimate content: %v", err)
	}

	// Wait for the miner to mine the block containing the legitimate post
	time.Sleep(20000 * time.Millisecond)

	// Attempt to duplicate the legitimate post
	posts, err := maliciousUser.ReadPosts()
	if err != nil {
		t.Fatalf("error reading user's posts: %v", err)
	}
	if len(posts) == 0 {
		t.Fatalf("user has no posts recorded")
	}
	lastPost := posts[len(posts)-1]
	lastPostEncoded := lastPost.EncodeBase64()
	lastPostJSON, _ := json.Marshal(lastPostEncoded)
	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/write", 3003), "application/json", bytes.NewBuffer(lastPostJSON))
	if err != nil {
		t.Fatalf("error replaying user's post: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status Bad Request for post duplication, but got %d", resp.StatusCode)
	}

	// Malicious user modifies a post's content without updating the signature correctly
	tamperedContent := "Tampered content"
	maliciousPost := blockchain.Post{
		User: &blockchain.GenerateKey().PublicKey, // New key simulating another user's identity or a new identity
		Body: blockchain.PostBody{
			Content:   tamperedContent,
			Timestamp: time.Now().UnixNano(),
		},
	}
	maliciousPost.Signature = blockchain.Sign(blockchain.GenerateKey(), maliciousPost.Body) // Incorrect signature
	maliciousPostEncoded := maliciousPost.EncodeBase64()
	maliciousPostJSON, _ := json.Marshal(maliciousPostEncoded)
	resp, err = http.Post(fmt.Sprintf("http://localhost:%d/write", 3003), "application/json", bytes.NewBuffer(maliciousPostJSON))
	if err != nil {
		t.Fatalf("error posting tampered content: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status Bad Request for tampered content, but got %d", resp.StatusCode)
	}

	// Check that only the legitimate post is on the blockchain
	posts, err = maliciousUser.ReadPosts()
	if err != nil {
		t.Fatalf("error when reading posts: %v", err)
	}
	if len(posts) != 1 || posts[0].Body.Content != legitimateContent {
		t.Fatalf("blockchain contains incorrect posts, expected 1 legitimate post, got %d or wrong content", len(posts))
	}
}

// TestMaliciousMiner - test whether the system rejects a block when a worker fakes or replays a user's post
func TestMaliciousMiner(t *testing.T) {
	tracker := Tracker.NewTracker(8080)
	tracker.Start()
	time.Sleep(1000 * time.Millisecond)

	// Create one legitimate miner
	miner := Miner.NewMiner(3000, 8080)
	miner.Start()
	// wait for everything to start
	time.Sleep(1000 * time.Millisecond)
	// post one message
	privateKey := blockchain.GenerateKey()
	post := blockchain.Post{
		User: &privateKey.PublicKey,
		Body: blockchain.PostBody{
			Content:   "Legitimate content",
			Timestamp: time.Now().UnixNano(),
		},
	}
	post.Signature = blockchain.Sign(privateKey, post.Body)
	postBase64 := post.EncodeBase64()
	postJSON, _ := json.Marshal(postBase64)
	resp, err := http.Post("http://localhost:3000/write", "application/json", bytes.NewReader(postJSON))
	if err != nil {
		t.Fatalf("error when writing blockchain: %v\n", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("error when writing blockchain: %v\n", err)
	}
	resp.Body.Close()
	// wait for a block to be mined
	time.Sleep(20000 * time.Millisecond)

	// tries to attack miner's /sync API with a fake post
	fakePost, _ := postBase64.DecodeBase64()
	fakePost.Body.Content = "Malicious content"
	fakePostBase64 := fakePost.EncodeBase64()
	syncReq := Miner.PostsJson{}
	syncReq.Posts = append(syncReq.Posts, fakePostBase64)
	fakePostJson, _ := json.Marshal(syncReq)
	resp, _ = http.Post("http://localhost:3000/sync", "application/json", bytes.NewReader(fakePostJson))
	resp.Body.Close()

	// tries to attack miner's /sync API with a replayed post
	resp, _ = http.Post("http://localhost:3000/sync", "application/json", bytes.NewReader(postJSON))
	resp.Body.Close()

	// tries to attack miner's /broadcast API with a very long, fake blockchain
	fakeBlockchain := make([]blockchain.BlockBase64, 100)
	fakeBroadcastReq := Miner.BlockChainJson{Blockchain: fakeBlockchain}
	fakeBroadcastJson, _ := json.Marshal(fakeBroadcastReq)
	resp, _ = http.Post("http://localhost:3000/broadcast", "application/json", bytes.NewReader(fakeBroadcastJson))
	resp.Body.Close()

	time.Sleep(10000 * time.Millisecond)
	user := User.NewUser(8080)
	posts, err := user.ReadPosts()
	if err != nil {
		t.Fatalf("error when reading posts: %v\n", err)
	}
	// should have only 1 legitimate post
	if len(posts) != 1 {
		t.Fatalf("wrong number of posts\n")
	}
	if posts[0].Body.Content != "Legitimate content" {
		t.Fatalf("wrong content of posts\n")
	}

	// clean up
	miner.Shutdown()
	tracker.Shutdown()
}
