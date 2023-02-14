package pkg

import (
	"context"
	"github.com/david-wiles/groupme-clone/internal"
	"github.com/gorilla/websocket"
)

type Client struct {
	Reads chan []byte
	internal.Serializer

	conn *websocket.Conn
}

func NewClient(ctx context.Context, conn *websocket.Conn) *Client {
	client := &Client{
		Reads:      make(chan []byte),
		Serializer: internal.JSONSerializer{},
		conn:       conn,
	}

	// Request for self-ID once socket is opened
	if err := conn.WriteMessage(websocket.TextMessage, []byte("whoami")); err != nil {
		panic(err)
	}

	go client.reader(ctx)

	return client
}

func (c *Client) reader(ctx context.Context) {
	defer c.close()

	messages := make(chan *internal.ClientMessage)

	go func() {
		for {
			_, bytes, err := c.conn.ReadMessage()
			if err != nil {
				return
			}

			decoded := &internal.ClientMessage{}
			if err := c.Deserialize(bytes, decoded); err != nil {
				continue
			}

			messages <- decoded
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-messages:
			// Add message payload to Reads
			c.Reads <- msg.Payload

			// Send ack
			if err := c.ackMessage(msg); err != nil {
				// Unable to ack message
			}
		}
	}
}

func (c *Client) ackMessage(msg *internal.ClientMessage) error {
	ack := internal.ClientAck{
		Cid: msg.Cid,
	}

	t, bytes, err := c.Serialize(ack)
	if err != nil {
		return err
	}
	if err := c.conn.WriteMessage(t, bytes); err != nil {
		return err
	}

	return nil
}

func (c *Client) close() {
	_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	_ = c.conn.Close()
}
