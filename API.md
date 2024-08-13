# Specification
## User
1. Ask a tracker for a miner.
2. Read all posts.
3. Write a post.

## Miner
1. Register itself to the tracker and get number of participants and up-to-date blockchain.
2. Miner needs to send heartbeats to the tracker.
3. Miner accepts write requests from users and adds them to its own post pool.
4. Miner syncs post pool and blockchain with (some of) other known miners.
5. Miner answers read request from a user.
6. Miner mines a new block and broadcasts it to all known miners and trackers.
7. Miner needs to answer other miner's broadcasts and updates its blockchain correspondingly.
8. Miner keeps track of all known miners from heartbeats.

## Tracker
1. Tracker answers a user request with a random miner.
2. Tracker answers register requests from miners and returns a list of all miners.
3. Tracker receives heartbeats as well from the registration API.

# API
## Tracker
### User requests a list of miners
**Command**: `/get_miners`

**Method**: `GET`

**Output**

**Code**: `200 OK`
```json
{
  "ports": [8080, 8081]
}
```
**Code**: `404 Not Found`

### A miner registers itself
**Command**: `/register`

**Method**: `POST`
```json
{
  "port": 8080
}
```

**Output**

**Code**: `200 OK`
```json
{
  "ports": [8080, 8081]
}
```

## Miner
### A user sends a read request
**Command**: `/read`

**Method**: `GET`

**Output**

**Code**: `200 OK`
```json
{
  "blockchain": []
}
```

### A user sends a write request
**Command**: `/write`

**Method**: `POST`
```json
{
  "user": "xlkdajfi1231n",
  "content": "Hello World",
  "timestamp": "0",
  "signature": "xlkdajfi1231n"
}
```

**Output**

**Code**: `200 OK`

### Another miner syncs with this miner
**Command**: `/sync`

**Method**: `POST`
```json
{
  "posts": []
}
```

**Output**

**Code**: `200 OK`

### Another miner wants to broadcast its new block
**Command**: `/broadcast`

**Method**: `POST`
```json
{
  "blockchain": []
}
```

**Output**

**Code**: `200 OK`