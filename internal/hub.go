package internal

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"sync"
	"time"
)

// Hub maintains a thread-safe map of UUID to WebsocketConnection. connMu guards the conns map,
// cidMu guards the inFlightCID map
type Hub struct {
	connMu sync.RWMutex
	conns  map[uuid.UUID]*WebsocketConnection

	// Map of CIDs to its channels used by a SendMessage call. When a client sends a message
	// with a CID matching one of these CIDs, the goroutine awaiting the acknowledgment will
	// be notified, ending that SendMessage call
	cidMu       sync.RWMutex
	inFlightCID map[uuid.UUID]chan struct{}

	Serializer
}

func NewHub() *Hub {
	return &Hub{
		connMu:      sync.RWMutex{},
		conns:       make(map[uuid.UUID]*WebsocketConnection),
		cidMu:       sync.RWMutex{},
		inFlightCID: make(map[uuid.UUID]chan struct{}),
		Serializer:  JSONSerializer{},
	}
}

// RegisterConnection will create a new WebsocketConnection from an underlying websocket.Conn
// and add it to the map of managed connections
func (hub *Hub) RegisterConnection(conn *websocket.Conn) *WebsocketConnection {
	ws := NewWebsocketConnection(conn, hub)

	hub.connMu.Lock()
	defer hub.connMu.Unlock()
	hub.conns[ws.ID] = ws

	return ws
}

// UnregisterConnection removes the WebsocketConnection identified by ID from the managed connections
func (hub *Hub) UnregisterConnection(ID uuid.UUID) {
	hub.connMu.Lock()
	defer hub.connMu.Unlock()
	delete(hub.conns, ID)
}

// UnsafeSendMessage attempts to send the message msg to the connection identified by ID. It will not
// attempt to verify that the client received the message, to wait for acknowledgement from the client use 'SendMessage'
func (hub *Hub) UnsafeSendMessage(ID uuid.UUID, msg Serializable) error {
	log.Printf("sending bytes to client uuid=%s", ID.String())

	hub.connMu.RLock()
	defer hub.connMu.RUnlock()

	if ws, ok := hub.conns[ID]; ok {
		select {
		case ws.Writes <- msg:
			// Successful write
			return nil
		default:
			// Error, client's write channel is full. The caller must implement retry logic
			// when using this function or propagate the error to the calling service
			return errors.New("buffer full")
		}
	} else {
		return errors.New("websocket not found")
	}
}

func (hub *Hub) SendMessage(ctx context.Context, ID uuid.UUID, msg []byte) error {
	// Format message using cid to wait for ack from client
	cid := uuid.New()
	clientMessage := &ClientMessage{
		Payload:     msg,
		Cid:         cid.String(),
		Acknowledge: true,
	}

	// Create channel for notification of ack
	ack := make(chan struct{})

	hub.cidMu.Lock()
	hub.inFlightCID[cid] = ack
	hub.cidMu.Unlock()

	// We always want to recover resources once this function exits, whether the ack was
	// successful or if the context expired
	defer func() {
		hub.cidMu.Lock()
		delete(hub.inFlightCID, cid)
		hub.cidMu.Unlock()
	}()

	// Add timeout for ack
	context.WithDeadline(ctx, time.Now().Add(10*time.Second))

	// Send message to client
	if err := hub.UnsafeSendMessage(ID, clientMessage); err != nil {
		return err
	}

	// Wait for ack from client
	select {
	case <-ack:
		// Success
		log.Printf("received ack cid=%s", cid.String())
		return nil
	case <-ctx.Done():
		// Context expired, return error to caller
		log.Printf("error waiting for client to acknowledge message. cid=%s err=%v", cid.String(), ctx.Err())
		return ctx.Err()
	}
}

func (hub *Hub) Acknowledge(cid string) error {
	parsedCID, err := uuid.Parse(cid)
	if err != nil {
		return err
	}

	hub.cidMu.RLock()
	defer hub.cidMu.RUnlock()

	if ack, ok := hub.inFlightCID[parsedCID]; ok {
		ack <- struct{}{}
	} else {
		return errors.New("key not found in map")
	}

	return nil
}
