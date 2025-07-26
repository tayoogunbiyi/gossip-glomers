package handlers

import (
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func RegisterAll(n *maelstrom.Node) {
	n.Handle("echo", EchoHandler(n))
	n.Handle("generate", GenerateHandler(n))
} 