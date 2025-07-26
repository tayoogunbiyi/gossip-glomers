package handlers

import (
	"encoding/json"
	"strconv"
	"sync"

	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

var BASE_TIMESTAMP = time.Date(2025, time.July, 22, 0, 0, 0, 0, time.UTC).UnixMilli()

var (
	idMutex        sync.Mutex
	lastTimestamp  int64
	counterInMs    int64
)


func GenerateHandler(n *maelstrom.Node) func(maelstrom.Message) error {
	return func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		body["type"] = "generate_ok"
		nodeID, err := strconv.ParseInt(n.ID()[1:], 10, 64)
		if err != nil {
			return err
		}
		body["id"] = NodeIDAndTimestampBasedID(nodeID)

		return n.Reply(msg, body)
	}
}


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

