package handlers

import (
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func register(n *maelstrom.Node, handler func(maelstrom.Message) error, commands ...string) {
	for _, command := range commands {
		n.Handle(command, handler)
	}
}

func RegisterAll(n *maelstrom.Node) {
	register(n, EchoHandler(n), "echo")
	register(n, GenerateHandler(n), "generate")
	register(n, SingleNodeBroadcastHandler(n), "broadcast", "topology", "read")
} 