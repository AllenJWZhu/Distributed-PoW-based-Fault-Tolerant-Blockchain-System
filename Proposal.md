# Proof-of-Work Blockchain Consensus System

## Overview
This document provides a comprehensive overview of the Proof-of-Work (PoW) blockchain consensus system. The system
consists of several components, including User, Worker, and Honest Tracker, which collaborate to maintain a
decentralized, secured, and consistent blockchain ledger.

## System Components
The implementation includes three roles: user, miner, and tracker. Users contact miners to post content to the
blockchain and read what other users have posted. Miners gather contents from users, mine a new block and attach the
contents to the blockchain. A new miner or a user can contact a tracker to obtain a list of currently active miners.
### User
A user sends an HTTP request to a worker to read or write posts. A user is uniquely identified by its public key.
### Miner
A worker runs automatically and uses HTTP to announce new blocks to its peers. When a new miner wants to join the
blockchain it requests information about other miners from the tracker. Miners need to periodically send heartbeats to
the tracker in order to let the tracker know they are active, and also to discover new miners.
### Tracker
A tracker runs automatically and answers requests from users and miners.
## Design Requirements and Specifications
This section contains specific details regarding the design and assumptions of our system.
## Assumptions
A tracker never lies to anyone.

In other cases anyone can lie to anyone, but the majority of the participants in this system should be well-behaved.
### User
- The User component serves as the interface for users to interact with the blockchain system.
- It allows users to read the current state of the blockchain and submit new content through HTTP requests.
- The User component validates and sanitizes the submitted content to ensure data integrity and security.
- After successful submission, the User component sends the content to one or more Miner components for processing.

### Miner
- Miner nodes participate in the block mining process and validate transactions to reach consensus on the state of the blockchain.
- When a new Miner node joins the network, it registers itself with the system, providing its network address.
- Miner nodes have read and write permissions to act on behalf of the User, allowing them to retrieve information and submit content.
- They obtain nonce values from the nonce host for block mining and can discover and connect to other Worker nodes in the network.
- Miner nodes select a subset of content from the pool, construct new blocks, and perform the proof-of-work algorithm to find valid block hashes.
- Once a valid block is found, the Miner submits the block to the network for validation and acceptance.
- Miner nodes also participate in the validation process by verifying the validity of blocks submitted by other Miner nodes.

### Tracker
- The Tracker always remains honest and helps maintain the integrity of the blockchain.
- When a new Miner node joins the network, it registers itself with the Tracker.
- The Tracker acknowledges the new Miner and adds it to the list of registered Miners.

## System Workflow

1. User Interaction:
   - Users interact with the blockchain system through the User component.
   - They can read the current state of the blockchain and submit new content using HTTP requests.

2. Content Submission and Broadcasting:
   - When a User submits new content, the User component sends it to one or more Miner nodes for processing.
   - The Miner nodes broadcast the content to other Miner nodes.

3. Block Mining and Broadcasting:
   - Miner nodes participate in the block mining process to create new blocks.
   - They select a subset of content from the pool, construct new blocks, and perform the proof-of-work algorithm to find valid block hashes.
   - When a Miner finds a valid block, it broadcasts the block to other Miners.

4. Block Validation and Acceptance:
   - Miner nodes receive broadcast blocks and validate them against their local copy of the blockchain.
   - If a block is valid, it is appended to the local blockchain, and the blockchain length is updated.

5. Miner Registration and Blockchain Length Information:
   - When a new Miner node joins the network, it registers itself with the system and the Tracker.

## Security and Trust
- The system incorporates security measures, such as authentication and encryption, to protect the communication between nodes.
- The Tracker serves as a trusted entity that helps maintain the integrity and consistency of the blockchain.
- However, relying on a single Tracker introduces a potential point of centralization and trust. So the amount of work a tracker should do is minimized.

## Programming Language:

We will use Go (Golang) as the primary programming language for implementing the Proof-of-Work blockchain consensus system.
Go provides excellent support for concurrent programming, networking, and cryptography, making it well-suited for blockchain development.

## Third-Party Libraries:

Gin Web Framework: A lightweight and fast web framework for building the HTTP endpoints and handling user requests.
github.com/emirpasic/gods/sets/treeset: A treeset implementation in Go.

## Test Cases
1. TestPostSafety(5): Test whether tampering a Post can be detected by signature.
2. TestBlockSafety(5): Test whether tampering a Block can be detected by signature.
3. TestMaliciousUser(7.5): Tests the behavior of miners when a malicious user tries to submit a tampered or duplicated post.
4. TestMaliciousMiner(7.5): Test whether a well-behaved miner rejects a block when another malicious miner fakes or replays a user's post.
5. TestMinerDiscovery(5): Test whether miners can register to the tracker and discover each other correctly.
6. TestNewUser(5): Tests the creation of a new user.
7. TestGetRandomMiners(5): Tests a user's retrieval of random miners from the tracker.
8. TestCompleteInteractions(10): Orchestrate complete interactions between a tracker, users and miners.
9. TestMergeBlockChainHeads(15): Test if the blockchain is resilient to network partition and multiple heads.
10. TestComputingPowerAttack(15): Simulate a successful computing power attack to a blockchain.

