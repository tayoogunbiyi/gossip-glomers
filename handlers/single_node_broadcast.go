package handlers

import (
	"encoding/json"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

var (
	messages = make(map[int]struct{})
	mu       sync.RWMutex
)

func SingleNodeBroadcastHandler(n *maelstrom.Node) func(maelstrom.Message) error {
	return func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		response := map[string]any{}

		switch msg.Type() {
		case "broadcast":
			response["type"] = "broadcast_ok"
			if messageValue, ok := body["message"]; ok {
				if messageInt, ok := messageValue.(float64); ok {
					mu.Lock()
					messages[int(messageInt)] = struct{}{}
					mu.Unlock()
				}
			}
		case "topology":
			response["type"] = "topology_ok"
		case "read":
			response["type"] = "read_ok"
			mu.RLock()
			msgs := make([]int, 0, len(messages))
			for m := range messages {
				msgs = append(msgs, m)
			}
			mu.RUnlock()
			response["messages"] = msgs
		}

		return n.Reply(msg, response)
	}
} 