package main

import (
	"log"
	"os"

	"maelstrom-echo/handlers"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()

	handlers.RegisterAll(n)

	log.Println("Starting node")
	if err := n.Run(); err != nil {
		log.Printf("ERROR: %s", err)
		os.Exit(1)
	}
}