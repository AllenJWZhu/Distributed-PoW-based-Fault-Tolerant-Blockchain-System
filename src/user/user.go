package user

import (
	"blockchain/blockchain"
	"blockchain/miner"
	"blockchain/tracker"
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/emirpasic/gods/sets/treeset"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"
)

// RWCount - Number of miners to select for writing posts
const RWCount = 3

// User represents a user in the blockchain system
type User struct {
	privateKey  *rsa.PrivateKey
	trackerPort int
}

// NewUser initializes a new instance of a User with a specific tracker port.
// The function generates a new RSA private key for the user and returns a User struct with the initialized values.
// Parameters:
//
//	trackerPort (int): The port number on which the tracker service is running.
//
// Returns:
//
//	*User: Pointer to the newly created User struct.
func NewUser(trackerPort int) *User {
	privateKey := blockchain.GenerateKey()
	return &User{
		privateKey:  privateKey,
		trackerPort: trackerPort,
	}
}

// GetRandomMiners retrieves a random subset of miners from the tracker service.
// It sends a GET request to the tracker's "/get_miners" endpoint and decodes the list of active miners.
// If the number of available miners is less than or equal to RWCount, it returns all miners. Otherwise, it shuffles
// the list and selects a random subset of RWCount miners.
// Returns:
//
//	([]int, error): A slice of selected miner ports and an error, if any occurred during the process.
func (u *User) GetRandomMiners() ([]int, error) {
	// Send a GET request to the tracker's "/get_miners" endpoint
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/get_miners", u.trackerPort))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to retrieve miners from the tracker")
	}

	// Decode the response body to get the list of miner ports
	var response tracker.PortsJson
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, errors.New("tracker sends invalid response")
	}
	ports := response.Ports

	// Select a random subset of miners
	if len(ports) <= RWCount {
		// If the number of miners is less than or equal to RWCount, use all miners
		return ports, nil
	}

	// Shuffle the miner ports randomly
	rand.Shuffle(len(ports), func(i, j int) {
		ports[i], ports[j] = ports[j], ports[i]
	})

	// Select the first RWCount miners from the shuffled list
	return ports[:RWCount], nil
}

// ReadPosts retrieves posts from a random subset of miners and consolidates them into a single, validated list.
// The function first retrieves a list of active miners and then concurrently fetches and decodes their stored blockchains.
// It verifies each blockchain's integrity and consistency, ensuring each block is valid and properly linked.
// Finally, it extracts and returns a de-duplicated list of posts sorted by their timestamp and user public key.
// Returns:
//
//	([]blockchain.Post, error): A slice of blockchain posts that have been validated and sorted, and an error, if any occurred.
func (u *User) ReadPosts() ([]blockchain.Post, error) {
	miners, err := u.GetRandomMiners()
	if err != nil {
		return nil, err
	}

	// send concurrent requests to get each miner's blockchain
	respChan := make(chan []blockchain.Block)
	for _, port := range miners {
		port := port
		go func(port int) {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/read", port))
			if err != nil {
				respChan <- nil
				return
			}
			defer resp.Body.Close()

			var respJson miner.BlockChainJson
			err = json.NewDecoder(resp.Body).Decode(&respJson)
			if err != nil {
				respChan <- nil
				return
			}
			// retrieve blockchain
			chain := make([]blockchain.Block, 0)
			for _, encoded := range respJson.Blockchain {
				decoded, err := encoded.DecodeBase64()
				if err != nil {
					respChan <- nil
					return
				}
				chain = append(chain, decoded)
			}
			respChan <- chain
		}(port)
	}
	chains := make([][]blockchain.Block, 0)
	for i := 0; i < len(miners); i++ {
		chains = append(chains, <-respChan)
	}
	// sort the chains from longest to shortest
	sort.Slice(chains, func(i, j int) bool {
		return len(chains[i]) > len(chains[j])
	})

	// find the first valid chain
	cmp := func(a, b any) int {
		post1 := a.(blockchain.Post)
		post2 := b.(blockchain.Post)
		if post1.Body.Timestamp != post2.Body.Timestamp {
			if post1.Body.Timestamp < post2.Body.Timestamp {
				return -1
			} else {
				return 1
			}
		}
		key1 := blockchain.PublicKeyToBytes(post1.User)
		key2 := blockchain.PublicKeyToBytes(post2.User)
		return bytes.Compare(key1, key2)
	}
	var posts *treeset.Set
VerifyChains:
	for _, chain := range chains {
		if len(chain) == 0 {
			continue VerifyChains
		}
		// each block must be valid
		for _, block := range chain {
			if !block.Verify() {
				continue VerifyChains
			}
		}
		// their hash value must form a chain
		if !bytes.Equal(chain[0].Header.PrevHash, make([]byte, 32)) {
			continue VerifyChains
		}
		for i := 1; i < len(chain); i++ {
			if !bytes.Equal(chain[i].Header.PrevHash, blockchain.Hash(chain[i-1].Header)) {
				continue VerifyChains
			}
		}
		// no duplicated posts
		posts = treeset.NewWith(cmp)
		for _, block := range chain {
			for _, post := range block.Posts {
				if posts.Contains(post) {
					posts = nil
					continue VerifyChains
				}
				posts.Add(post)
			}
		}
		// done
		break
	}
	if posts == nil {
		return nil, errors.New("failed to receive a valid blockchain")
	}
	postsList := make([]blockchain.Post, 0)
	iter := posts.Iterator()
	for iter.Next() {
		postsList = append(postsList, iter.Value().(blockchain.Post))
	}
	return postsList, nil
}

// WritePost creates and signs a new post with the user's private key, then concurrently sends it to a subset of miners.
// It generates a new post using the provided content and current timestamp, signs it, and encodes it in base64 format.
// The function then retrieves a list of active miners and sends the post to each via a POST request.
// It waits for all requests to complete and checks for errors, returning the first encountered error.
// Parameters:
//
//	content (string): The content of the post to be created.
//
// Returns:
//
//	error: An error if any occurred during the process of writing the post.
func (u *User) WritePost(content string) error {
	// Create a new post with the given content and the user's public key
	post := blockchain.Post{
		User: &u.privateKey.PublicKey,
		Body: blockchain.PostBody{
			Content:   content,
			Timestamp: time.Now().UnixNano(),
		},
	}

	// Sign the post using the user's private key
	post.Signature = blockchain.Sign(u.privateKey, post.Body)

	// Encode the post to base64
	postBase64 := post.EncodeBase64()

	// Determine the number of miners to use
	miners, err := u.GetRandomMiners()
	if err != nil {
		return err
	}

	// Create a wait group to wait for concurrent requests to finish
	var wg sync.WaitGroup
	errChan := make(chan error, len(miners)) // Channel to collect errors

	// Send POST requests to the selected miners concurrently
	for _, port := range miners {
		port := port
		wg.Add(1)
		go func(port int) {
			defer wg.Done()

			// Send a POST request to the miner's "/write" endpoint with the post data
			postJSON, _ := json.Marshal(postBase64)
			resp, err := http.Post(fmt.Sprintf("http://localhost:%d/write", port), "application/json", bytes.NewReader(postJSON))
			if err != nil {
				errChan <- err
				return
			}
			if resp.StatusCode != http.StatusOK {
				errChan <- fmt.Errorf("miner rejected post: status code %d", resp.StatusCode)
			}
			resp.Body.Close()
		}(port)
	}

	// Wait for all concurrent requests to finish
	wg.Wait()
	close(errChan) // Close channel to finish range iteration

	// Check for errors from the error channel
	for e := range errChan {
		if e != nil {
			return e // Return the first error encountered
		}
	}

	return nil
}
