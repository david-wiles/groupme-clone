package main

import (
	"context"
	"flag"
	"github.com/david-wiles/groupme-clone/pkg"
	"github.com/gorilla/websocket"
	"log"
)

// Creates a virtual device which will act as a chat client
func main() {

	u := flag.String("url", "ws://localhost:8080", "Set the URL of the chat server")

	flag.Parse()

	conn, _, err := websocket.DefaultDialer.Dial(*u, nil)
	if err != nil {
		panic(err)
		return
	}

	client := pkg.NewClient(context.Background(), conn)
	for msg := range client.Reads {
		if message, ok := msg.(*pkg.ClientMessage); ok {
			log.Printf("%s", string(message.Payload))
		}
	}
}
