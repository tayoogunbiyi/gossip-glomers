# Gossip Gloomers - Distributed Systems Solutions

This repository contains my solutions to the [Gossip Glomers](https://fly.io/dist-sys/) distributed systems challenges by Fly.io, built on top of [Maelstrom](https://github.com/jepsen-io/maelstrom) - Kyle Kingsbury's distributed systems testing platform.


## Challenge #1: Echo

A simple "hello world" for distributed systems - implement a node that echoes back any message it receives. This challenge helps understand the Maelstrom message format and basic request/response patterns

## Challenge #2: Unique ID Generation - Deep Dive

### The Problem

[Challenge #2](https://fly.io/dist-sys/2/) requires implementing a globally-unique ID generation system that remains totally available during network partitions while handling high throughput across multiple nodes.

### My Solution: Snowflake-Inspired Approach

I implemented a **timestamp + node ID + counter** approach similar to Twitter's Snowflake algorithm:

```go
func NodeIDAndTimestampBasedID(nodeID int64) int64 {
    idMutex.Lock()
    defer idMutex.Unlock()

    timestamp := time.Now().UnixMilli() - BASE_TIMESTAMP
    if timestamp != lastTimestamp {
        lastTimestamp = timestamp
        counterInMs = 0
    } else {
        counterInMs++
    }
    return (timestamp << 20) | (nodeID << 10) | counterInMs
}
```

#### Bit Layout (64-bit total):
- **44 bits**: Timestamp (milliseconds since July 22, 2025 base)
- **10 bits**: Node ID (extracted from Maelstrom node identifier)
- **10 bits**: Counter (for IDs generated within the same millisecond)

#### Key Features:
This approach guarantees uniqueness through timestamp + node ID + counter combination while providing high throughput (1,024 IDs per millisecond per node), time-ordering, zero coordination, and thread-safety.

## Alternative Approaches

**UUIDv7**: Latest time-sortable UUID standard with 48-bit timestamp and random components

**ULID**: 128-bit identifier with 48-bit timestamp + 80-bit randomness


## Challenge #3a: Single-Node Broadcast

[Challenge #3a](https://fly.io/dist-sys/3a/) introduces a multi-part challenge to build a fault-tolerant, multi-node broadcast system. This first part establishes the foundation by implementing a broadcast service that runs on a single node.

### The Problem

The task is to create a durable, in-memory store on a single node that handles `broadcast`, `read`, and `topology` RPCs. The node must store integers from `broadcast` messages and return them on `read` requests.

### My Solution: Thread-Safe, In-Memory Broadcast

Use a `map[int]struct{}` to store a unique set of messages. To handle concurrent requests safely and efficiently, access to the map is controlled by a `sync.RWMutex`, which allows for concurrent reads.

```go
func SingleNodeBroadcastHandler(n *maelstrom.Node) func(maelstrom.Message) error {
	return func(msg maelstrom.Message) error {
		// ... unmarshal body ...
		switch msg.Type() {
		case "broadcast":
			// ...
			mu.Lock() // Exclusive lock for writing
			messages[int(messageInt)] = struct{}{}
			mu.Unlock()
			// ...
		case "read":
			// ...
			mu.RLock() // Read lock allows concurrent reads
			msgs := make([]int, 0, len(messages))
			for m := range messages {
				msgs = append(msgs, m)
			}
			mu.RUnlock()
			response["messages"] = msgs
		}
		// ...
	}
}
```

## Challenge #3b: Multi-Node Broadcast

[Challenge #3b](https://fly.io/dist-sys/3b/) extends the single-node broadcast to a multi-node cluster where messages must propagate efficiently across all nodes without network partitions.

### The Problem

Build on the single-node implementation to propagate broadcast values across a 5-node cluster. Messages should reach all nodes within a few seconds while avoiding inefficient approaches like sending entire datasets on every message.

### My Solution: Topology-Aware Gossip Protocol

Implemented an efficient gossip-based broadcast that uses the network topology to minimize redundant transmissions and prevent infinite message loops.

```go
func sendGossipToNodes(n *maelstrom.Node, targetNodes []string, messageValue interface{}) {
	for _, node := range targetNodes {
		go func(targetNode string) {
			n.RPC(targetNode, map[string]any{
				"type":    "broadcast", 
				"message": messageValue,
			}, func(msg maelstrom.Message) error {
				return nil
			})
		}(node)
	}
}

func BroadcastHandler(n *maelstrom.Node) func(maelstrom.Message) error {
	// ... check if message already seen ...
	if !alreadySeen {
		neighbors := getNeighbors(n.ID())
		filteredNeighbors := make([]string, 0, len(neighbors))
		for _, neighbor := range neighbors {
			if neighbor != msg.Src { // Prevent loops
				filteredNeighbors = append(filteredNeighbors, neighbor)
			}
		}
		sendGossipToNodes(n, filteredNeighbors, message)
	}
}
```



## Running the Tests

```bash
# Build the binary
go build -o gossip-gloomers

# Run Challenge #1 tests
./maelstrom/maelstrom test -w echo --bin ./gossip-gloomers --node-count 1 --time-limit 10

# Run Challenge #2 tests
./maelstrom/maelstrom test -w unique-ids --bin ./gossip-gloomers --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition

# Run Challenge #3a tests
./maelstrom test -w broadcast --bin ./gossip-gloomers --node-count 1 --time-limit 20 --rate 10

# Run Challenge #3b tests
./maelstrom test -w broadcast --bin ./gossip-gloomers --node-count 5 --time-limit 20 --rate 10
```
