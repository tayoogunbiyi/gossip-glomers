package handlers

import (
	"encoding/json"
	"sort"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

var (
	messages = make(map[int]struct{})
	mu       sync.RWMutex
	topology map[string][]string
)

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

func getNeighbors(nodeID string) []string {
	if topology == nil {
		return nil
	}
	return topology[nodeID]
}

func BroadcastHandler(n *maelstrom.Node) func(maelstrom.Message) error {
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
					message := int(messageInt)
					
					mu.RLock()
					_, alreadySeen := messages[message]
					mu.RUnlock()
					
					if alreadySeen {
						return n.Reply(msg, response)
					}
					
					mu.Lock()
					messages[message] = struct{}{}
					mu.Unlock()
					
					neighbors := getNeighbors(n.ID())
					filteredNeighbors := make([]string, 0, len(neighbors))
					for _, neighbor := range neighbors {
						if neighbor != msg.Src {
							filteredNeighbors = append(filteredNeighbors, neighbor)
						}
					}
					sendGossipToNodes(n, filteredNeighbors, message)
				}
			}
		case "topology":
			response["type"] = "topology_ok"
			if topologyData, ok := body["topology"]; ok {
				if topologyMap, ok := topologyData.(map[string]any); ok {
					topology = make(map[string][]string)
					for nodeID, neighborsAny := range topologyMap {
						if neighborsSlice, ok := neighborsAny.([]any); ok {
							neighbors := make([]string, 0, len(neighborsSlice))
							for _, neighbor := range neighborsSlice {
								if neighborStr, ok := neighbor.(string); ok {
									neighbors = append(neighbors, neighborStr)
								}
							}
							topology[nodeID] = neighbors
						}
					}
				}
			}
		case "read":
			response["type"] = "read_ok"
			mu.RLock()
			msgs := make([]int, 0, len(messages))
			for m := range messages {
				msgs = append(msgs, m)
			}
			mu.RUnlock()
			sort.Ints(msgs)
			response["messages"] = msgs
		}
		return n.Reply(msg, response)
	}
} 