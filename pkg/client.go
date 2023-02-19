package pkg

import (
	"context"
	"github.com/gorilla/websocket"
)

type Client struct {
	Reads   chan Serializable
	Webhook string
	Serializer

	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc
}

func NewClient(ctx context.Context, conn *websocket.Conn) (*Client, error) {
	serializer := JSONSerializer{}

	_, bytes, err := conn.ReadMessage()
	if err != nil {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		_ = conn.Close()
		return nil, err
	}

	whoami := WhoAmIResponse{}
	if err := serializer.Deserialize(bytes, &whoami); err != nil {
		return nil, err
	}

	clientContext, cancel := context.WithCancel(ctx)

	client := &Client{
		Reads:      make(chan Serializable),
		Webhook:    whoami.Webhook,
		Serializer: JSONSerializer{},
		conn:       conn,
		ctx:        clientContext,
		cancel:     cancel,
	}

	go client.reader()

	return client, nil
}

func (c *Client) Close() {
	c.cancel()
}

func (c *Client) reader() {
	defer c.close()

	messages := make(chan *ClientMessage)

	go func() {
		for {
			_, bytes, err := c.conn.ReadMessage()
			if err != nil {
				return
			}

			decoded := &ClientMessage{}
			if err := c.Deserialize(bytes, decoded); err != nil {
				continue
			}

			messages <- decoded
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-messages:
			// Add message payload to Reads
			c.Reads <- msg

			// Send ack
			if err := c.ackMessage(msg); err != nil {
				// Unable to ack message
			}
		}
	}
}

func (c *Client) ackMessage(msg *ClientMessage) error {
	ack := ClientAck{
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
	close(c.Reads)
	_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	_ = c.conn.Close()
}
