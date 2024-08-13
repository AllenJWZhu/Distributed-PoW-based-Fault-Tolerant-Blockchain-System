package tests

import (
	"blockchain/blockchain"
	Miner "blockchain/miner"
	Tracker "blockchain/tracker"
	User "blockchain/user"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"testing"
	"time"
)

// TestCompleteInteractions - orchestrate complete interactions between a tracker, users and miners
func TestCompleteInteractions(t *testing.T) {
	tracker := Tracker.NewTracker(8080)
	tracker.Start()
	time.Sleep(1000 * time.Millisecond)
	// register 6 miners
	miners := make([]*Miner.Miner, 0)
	for i := 0; i < 6; i++ {
		miner := Miner.NewMiner(3000+i, 8080)
		miner.Start()
		miners = append(miners, miner)
	}
	// register 6 users
	users := make([]*User.User, 0)
	for i := 0; i < 6; i++ {
		users = append(users, User.NewUser(8080))
	}
	// wait for everything to be ready
	time.Sleep(500 * time.Millisecond)

	// each user posts something
	for i := 0; i < 6; i++ {
		err := users[i].WritePost(fmt.Sprintf("Hello world from %d", i))
		if err != nil {
			t.Fatalf("error when posting: %v", err)
		}
	}

	// wait for the blockchain to reach consensus
	time.Sleep(20000 * time.Millisecond)
	posts, err := users[0].ReadPosts()
	if err != nil {
		t.Fatalf("error when reading: %v", err)
	}
	if len(posts) != 6 {
		t.Fatalf("not enough posts on the blockchain")
	}
	for i := 0; i < 6; i++ {
		if posts[i].Body.Content != fmt.Sprintf("Hello world from %d", i) {
			t.Fatalf("wrong body for post %d: %s", i, posts[i].Body.Content)
		}
	}
	// gracefully shutdown everything
	for _, miner := range miners {
		miner.Shutdown()
	}
	tracker.Shutdown()
}

// TestMergeBlockChainHeads - Test if the blockchain is resilient to network partition and multiple heads.
// Separate heads of the same chain can be created when network is partitioned, or multiple miners submit and broadcast
// their new blocks at roughly the same time (which is hard to simulate but handled identically as the previous case).
// The test first posts 2 messages to the blockchain.
// Then partition the network manually, creating 2 heads, and post 1 message to each head.
// Finally, re-merge the network. The longer head should defeat the shorter head, and that one message posted to the
// shorter head must return to the pool. No post should be lost in this process.
func TestMergeBlockChainHeads(t *testing.T) {
	tracker := NewPartitionTracker(8080)
	tracker.Start()
	time.Sleep(1000 * time.Millisecond)

	// register 10 miners
	miners := make([]*Miner.Miner, 0)
	for i := 0; i < 10; i++ {
		miner := Miner.NewMiner(3000+i, 8080)
		miner.Start()
		miners = append(miners, miner)
	}
	// post two messages, and then let miners mine for some time
	err := WriteBlockchain(3000, "Hello from 0")
	if err != nil {
		t.Fatalf("failed to write to miner 3000: %v", err)
	}
	err = WriteBlockchain(3001, "Hello from 1")
	if err != nil {
		t.Fatalf("failed to write to miner 3001: %v", err)
	}
	time.Sleep(20000 * time.Millisecond)
	// they should reach a consensus
	chain1 := ReadBlockchain(3000)
	if len(chain1) == 0 {
		t.Fatalf("failed to retrieve from miner 3000")
	}
	chain2 := ReadBlockchain(3001)
	if len(chain2) == 0 {
		t.Fatalf("failed to retrieve from miner 3001")
	}
	if len(chain1) != len(chain2) || !reflect.DeepEqual(chain1, chain2) {
		t.Fatalf("failed to reach a consensus")
	}

	// partition the network, and post one message to each head
	t.Log("Partitioning the network...")
	tracker.Partition(true)
	time.Sleep(1000 * time.Millisecond)
	err = WriteBlockchain(3002, "Hello from 2")
	if err != nil {
		t.Fatalf("failed to write to miner 3002: %v", err)
	}
	err = WriteBlockchain(3003, "Hello from 3")
	if err != nil {
		t.Fatalf("failed to write to miner 3003: %v", err)
	}
	time.Sleep(19000 * time.Millisecond)
	chain1 = ReadBlockchain(3000)
	if len(chain1) == 0 {
		t.Fatalf("failed to retrieve from miner 3000")
	}
	chain2 = ReadBlockchain(3001)
	if len(chain2) == 0 {
		t.Fatalf("failed to retrieve from miner 3001")
	}
	if reflect.DeepEqual(chain1, chain2) {
		t.Fatalf("failed to create two forks of a blockchain")
	}

	// merge the network again, and post two messages
	t.Log("Re-merging the network...")
	tracker.Partition(false)
	err = WriteBlockchain(3004, "Hello from 4")
	if err != nil {
		t.Fatalf("failed to write to miner 3004: %v", err)
	}
	err = WriteBlockchain(3005, "Hello from 5")
	if err != nil {
		t.Fatalf("failed to write to miner 3005: %v", err)
	}
	time.Sleep(20000 * time.Millisecond)
	// they should reach a consensus
	chain1 = ReadBlockchain(3000)
	if len(chain1) == 0 {
		t.Fatalf("failed to retrieve from miner 3000")
	}
	chain2 = ReadBlockchain(3001)
	if len(chain2) == 0 {
		t.Fatalf("failed to retrieve from miner 3001")
	}
	if len(chain1) != len(chain2) || !reflect.DeepEqual(chain1, chain2) {
		t.Fatalf("failed to reach a consensus")
	}
	user := User.NewUser(8080)
	posts, _ := user.ReadPosts()
	if len(posts) != 6 {
		t.Fatalf("wrong number of posts")
	}
	for i := 0; i < 6; i++ {
		if posts[i].Body.Content != fmt.Sprintf("Hello from %d", i) {
			t.Fatalf("wrong content of posts")
		}
	}

	// cleanup
	for _, miner := range miners {
		miner.Shutdown()
	}
	tracker.Shutdown()
}

// TestComputingPowerAttack - Simulate a successful computing power attack to a blockchain.
// First 6 miners are in the system.
// After 5 seconds, a malicious miner with 4 goroutines start attacking. This should not be successful.
// After 10 seconds, all but 1 miner are shut down. Now the malicious miner should be able to out-compute well-behaved
// miners.
// After 50 seconds, the blockchain should have been attacked successfully.
func TestComputingPowerAttack(t *testing.T) {
	tracker := Tracker.NewTracker(8080)
	tracker.Start()
	time.Sleep(1000 * time.Millisecond)

	user := User.NewUser(8080)
	// set up 6 well-behaved miners
	miners := make([]*Miner.Miner, 0)
	for i := 0; i < 6; i++ {
		miner := Miner.NewMiner(3000+i, 8080)
		miner.Start()
		miners = append(miners, miner)
	}
	// let them mine for 5 seconds
	time.Sleep(5000 * time.Millisecond)

	// malicious miner tries to create a branch on top of this blockchain
	quit := make(chan bool)
	go func() {
		attackChain := make([]blockchain.Block, 0)
		for {
			// set up an attack block
			privateKey := blockchain.GenerateKey()
			attackPost := blockchain.Post{
				User:      &privateKey.PublicKey,
				Signature: nil,
				Body: blockchain.PostBody{
					Content:   "Spam",
					Timestamp: time.Now().UnixNano(),
				},
			}
			attackPost.Signature = blockchain.Sign(privateKey, attackPost.Body)
			posts := make([]blockchain.Post, 0)
			posts = append(posts, attackPost)
			block := blockchain.Block{
				Header: blockchain.BlockHeader{
					PrevHash:  make([]byte, 32),
					Summary:   blockchain.Hash(posts),
					Timestamp: time.Now().UnixNano(),
				},
				Posts: posts,
			}
			if len(attackChain) > 0 {
				hash := blockchain.Hash(attackChain[len(attackChain)-1].Header)
				copy(block.Header.PrevHash, hash)
			}
			success := false
			// mine this attack block
			for !success {
				select {
				case <-quit:
					quit <- true
					return
				default:
					break
				}
				// use 4 goroutines for computing
				chanNonce := make(chan uint32)
				nonces := make([]uint32, 0)
				for i := 0; i < 4; i++ {
					go func() {
						// create a local copy
						encoded := block.EncodeBase64()
						block, _ := encoded.DecodeBase64()
					MineIter:
						for i := 0; i < 10000; i++ {
							block.Header.Nonce = rand.Uint32()
							hash := blockchain.Hash(block.Header)
							zeroBytes := blockchain.TARGET / 8
							zeroBits := blockchain.TARGET % 8
							// the first zeroBytes bytes of hash must be zero
							for i := 0; i < zeroBytes; i++ {
								if hash[i] != 0 {
									continue MineIter
								}
							}
							// and then zeroBits bits of hash must be zero
							if zeroBits > 0 {
								nextByte := hash[zeroBytes]
								nextByte = nextByte >> (8 - zeroBits)
								if nextByte != 0 {
									continue MineIter
								}
							}
							chanNonce <- block.Header.Nonce
							return
						}
						chanNonce <- 0
					}()
				}
				for i := 0; i < 4; i++ {
					result := <-chanNonce
					nonces = append(nonces, result)
					if result != 0 {
						block.Header.Nonce = result
						success = true
					}
				}
			}
			// success
			attackChain = append(attackChain, block)
			encodedChain := make([]blockchain.BlockBase64, 0)
			for _, b := range attackChain {
				encodedChain = append(encodedChain, b.EncodeBase64())
			}
			log.Printf("Attack chain has length of %d\n", len(attackChain))
			// broadcast
			for i := 0; i < 6; i++ {
				request := Miner.BlockChainJson{Blockchain: encodedChain}
				reqJson, _ := json.Marshal(request)
				resp, _ := http.Post(
					fmt.Sprintf("http://localhost:%d/broadcast", 3000+i),
					"application/json", bytes.NewReader(reqJson))
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}
			}
		}
	}()
	t.Log("Started malicious miners")

	// well-behaved miners should be able to out-compute malicious miners
	time.Sleep(10000 * time.Millisecond)
	posts, err := user.ReadPosts()
	if err != nil {
		t.Fatalf("error when reading posts: %v\n", err)
	}
	if len(posts) != 0 {
		t.Fatalf("blockchain is attacked by malicious miners\n")
	}

	// now all but 1 miner are shut down
	for i := 1; i < 6; i++ {
		miners[i].Shutdown()
	}
	t.Log("Shut down 5 miners")
	// now the malicious miner should out-compute well-behaved miners
	time.Sleep(50000 * time.Millisecond)
	posts, err = user.ReadPosts()
	if err != nil {
		t.Fatalf("error when reading posts: %v\n", err)
	}
	if len(posts) == 0 {
		t.Fatalf("malicious miners did not out-compute well-behaved miners\n")
	}

	// clean up
	for i := 0; i < 1; i++ {
		miners[i].Shutdown()
	}
	quit <- true
	<-quit
	tracker.Shutdown()
}
