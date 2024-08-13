package tests

import (
	"blockchain/blockchain"
	"crypto/rsa"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// TestPostSafety verifies the integrity and security mechanisms of blockchain posts.
// The test checks three key aspects:
// 1. Signature Validation: Ensures that a post with a valid signature verifies correctly.
// 2. Encoding and Decoding: Confirms that a post remains unchanged through encode/decode cycles.
// 3. Tamper Detection: Tests the system's ability to detect and reject tampered posts after signing.
// It performs tampering on both content and timestamp to check the robustness of the signature system.
func TestPostSafety(t *testing.T) {
	privateKey := blockchain.GenerateKey()
	post := blockchain.Post{
		User: &privateKey.PublicKey,
		Body: blockchain.PostBody{
			Content:   "Hello World",
			Timestamp: time.Now().UnixNano(),
		},
	}
	post.Signature = blockchain.Sign(privateKey, post.Body)
	if !post.Verify() {
		t.Fatal("Body is not signed correctly")
	}

	// encoding and then decoding should return the identical block
	encoded := post.EncodeBase64()
	decoded, _ := encoded.DecodeBase64()
	if !reflect.DeepEqual(post, decoded) {
		t.Fatal("post is not encoded or decoded correctly")
	}

	// tamper the content of post
	post.Body.Content = "Bye World"
	if post.Verify() {
		t.Fatal("signature fails to detect a tamper of content")
	}

	// tamper the timestamp of post
	post.Body.Content = "Hello World"
	post.Body.Timestamp = time.Now().UnixNano()
	if post.Verify() {
		t.Fatal("signature fails to detect a tamper of content")
	}
}

// TestBlockSafety evaluates the blockchain's ability to detect tampering within its blocks.
// The test examines:
// 1. Block Validation: Ensures that a freshly mined block with correct signatures and hashes is valid.
// 2. Encode and Decode Integrity: Confirms that a block's structure is maintained correctly through encoding and decoding cycles.
// 3. Tamper Detection: Tests detection of tampering in block contents, including post deletions and modifications to the 'PrevHash'.
// This function performs detailed checks by modifying block components and verifying that these changes invalidate the block.
func TestBlockSafety(t *testing.T) {
	users := make([]*rsa.PrivateKey, 0)
	posts := make([]blockchain.Post, 0)
	for i := 0; i < 3; i++ {
		privateKey := blockchain.GenerateKey()
		post := blockchain.Post{
			User: &privateKey.PublicKey,
			Body: blockchain.PostBody{
				Content:   fmt.Sprintf("Hello from %d", i),
				Timestamp: time.Now().UnixNano(),
			},
		}
		post.Signature = blockchain.Sign(privateKey, post.Body)
		users = append(users, privateKey)
		posts = append(posts, post)
	}
	block := blockchain.Block{
		Header: blockchain.BlockHeader{
			PrevHash:  make([]byte, 32),
			Summary:   blockchain.Hash(posts),
			Timestamp: time.Now().UnixNano(),
		},
		Posts: posts,
	}
	start := time.Now().UnixMilli()
	count := 0
mine:
	for {
		count++
		block.Header.Nonce = rand.Uint32()
		hash := blockchain.Hash(block.Header)
		zeroBytes := blockchain.TARGET / 8
		zeroBits := blockchain.TARGET % 8
		// the first zeroBytes bytes of hash must be zero
		for i := 0; i < zeroBytes; i++ {
			if hash[i] != 0 {
				continue mine
			}
		}
		// and then zeroBits bits of hash must be zero
		if zeroBits > 0 {
			nextByte := hash[zeroBytes]
			nextByte = nextByte >> (8 - zeroBits)
			if nextByte != 0 {
				continue mine
			}
		}
		break
	}
	end := time.Now().UnixMilli()
	t.Logf("used %d ms (%d iterations) to mine a block", end-start, count)

	if !block.Verify() {
		t.Fatalf("the mined block is not valid")
	}

	// encoding and then decoding should return the identical block
	encoded := block.EncodeBase64()
	decoded, _ := encoded.DecodeBase64()
	if !reflect.DeepEqual(block, decoded) {
		t.Fatal("block is not encoded or decoded correctly")
	}

	// delete a post
	block.Posts = posts[:2]
	if block.Verify() {
		t.Fatalf("fails to detect a tamper of posts")
	}

	// tamper PrevHash
	block.Header.PrevHash[0] = 1
	if block.Verify() {
		t.Fatalf("fails to detect a tamper of previous block's hash")
	}
}
