# Maelstrom Echo - Gossip Glomers Solutions

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



## Running the Tests

```bash
# Build the binary
go build -o maelstrom-echo

# Run Challenge #1 tests
./maelstrom/maelstrom test -w echo --bin ./maelstrom-echo --node-count 1 --time-limit 10

# Run Challenge #2 tests
./maelstrom/maelstrom test -w unique-ids --bin ./maelstrom-echo --time-limit 30 --rate 1000 --node-count 3 --availability total --nemesis partition
```
