package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

const (
	RPC_TIMEOUT = 5 * time.Second
	MAX_RETRIES = 10
)

var (
	messages = make(map[int]struct{})
	mu       sync.RWMutex
	topology map[string][]string
	tracker  = &retryTracker{retries: make(map[string]context.CancelFunc)}
)

type retryTracker struct {
	mu      sync.Mutex
	retries map[string]context.CancelFunc
}

func (rt *retryTracker) storeCancelFunc(key string, cancel context.CancelFunc) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.retries[key] = cancel
}

func (rt *retryTracker) cancelAndRemove(key string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if cancel, exists := rt.retries[key]; exists {
		cancel()
		delete(rt.retries, key)
	}
}

func (rt *retryTracker) removeCancelFunc(key string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.retries, key)
}

func generateMessageID(messageValue interface{}) string {
	return fmt.Sprintf("%v", messageValue)
}



func createTimeoutContext(retryKey string, onTimeout func()) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), RPC_TIMEOUT)
	tracker.storeCancelFunc(retryKey, cancel)
	
	go handleTimeout(ctx, retryKey, onTimeout)
	
	return ctx, cancel
}

func handleTimeout(ctx context.Context, retryKey string, onTimeout func()) {
	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		tracker.removeCancelFunc(retryKey)
		onTimeout()
	}
}

func createSuccessCallback(retryKey string, cancel context.CancelFunc) func(maelstrom.Message) error {
	return func(msg maelstrom.Message) error {
		tracker.cancelAndRemove(retryKey)
		cancel()
		return nil
	}
}

func handleRPCError(retryKey string, cancel context.CancelFunc, onError func()) {
	tracker.removeCancelFunc(retryKey)
	cancel()
	onError()
}

func executeRPC(n *maelstrom.Node, targetNode string, messageValue interface{}, retryKey string, cancel context.CancelFunc, onError func()) {
	err := n.RPC(targetNode, map[string]any{
		"type":    "broadcast",
		"message": messageValue,
	}, createSuccessCallback(retryKey, cancel))
	
	if err != nil {
		handleRPCError(retryKey, cancel, onError)
	}
}

func sendWithRetry(n *maelstrom.Node, targetNode string, messageValue interface{}, messageID string, attemptsLeft int) {
	if attemptsLeft <= 0 {
		return
	}
	
	retryKey := fmt.Sprintf("%s:%s", targetNode, messageID)
	
	_, cancel := createTimeoutContext(retryKey, func() {
		sendWithRetry(n, targetNode, messageValue, messageID, attemptsLeft-1)
	})
	
	executeRPC(n, targetNode, messageValue, retryKey, cancel, func() {
		sendWithRetry(n, targetNode, messageValue, messageID, attemptsLeft-1)
	})
}

func sendGossipToNodes(n *maelstrom.Node, targetNodes []string, messageValue interface{}) {
	messageID := generateMessageID(messageValue)
	
	for _, node := range targetNodes {
		go sendWithRetry(n, node, messageValue, messageID, MAX_RETRIES)
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