package internal

import (
	"context"
	"errors"
	"github.com/david-wiles/groupme-clone/pkg"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

// Hub maintains a thread-safe map of UUID to WebsocketConnection. connMu guards the conns map,
// cidMu guards the inFlightCID map
type Hub struct {
	connMu sync.RWMutex
	conns  map[uuid.UUID]*WebsocketConnection

	RegistrationEngine

	// Map of CIDs to its channels used by a SendMessage call. When a client sends a message
	// with a CID matching one of these CIDs, the goroutine awaiting the acknowledgment will
	// be notified, ending that SendMessage call
	cidMu       sync.RWMutex
	inFlightCID map[uuid.UUID]chan struct{}

	hostname string

	pkg.Serializer
}

func NewHub(hostname string, engine RegistrationEngine) *Hub {
	return &Hub{
		connMu:             sync.RWMutex{},
		conns:              make(map[uuid.UUID]*WebsocketConnection),
		cidMu:              sync.RWMutex{},
		inFlightCID:        make(map[uuid.UUID]chan struct{}),
		hostname:           hostname,
		Serializer:         pkg.JSONSerializer{},
		RegistrationEngine: engine,
	}
}

// RegisterConnection will create a new WebsocketConnection from an underlying websocket.Conn
// and add it to the map of managed connections
func (hub *Hub) RegisterConnection(userID uuid.UUID, conn *websocket.Conn) *WebsocketConnection {
	ws := NewWebsocketConnection(conn, hub, userID)

	hub.connMu.Lock()
	hub.conns[ws.ID] = ws
	hub.connMu.Unlock()

	webhook := hub.hostname + "/" + ws.ID.String()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Register the client's webhook
	if err := hub.SetUserWebhook(ctx, webhook, userID); err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"userID": userID,
			"wsID":   ws.ID,
		}).Errorln("unable to set client webhook")
	}

	return ws
}

// UnregisterConnection removes the WebsocketConnection identified by ID from the managed connections
func (hub *Hub) UnregisterConnection(wsID uuid.UUID) {
	var userID *uuid.UUID

	hub.connMu.Lock()
	ws, ok := hub.conns[wsID]
	if ok {
		userID = &ws.userID
	}

	delete(hub.conns, wsID)
	hub.connMu.Unlock()

	if userID != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Register the client's webhook
		if err := hub.RemoveUserWebhook(ctx, *userID); err != nil {
			log.WithFields(log.Fields{
				"err":    err,
				"userID": userID,
				"wsID":   ws.ID,
			}).Errorln("unable to set client webhook")
		}
	}
}

// UnsafeSendMessage attempts to send the message msg to the connection identified by ID. It will not
// attempt to verify that the client received the message, to wait for acknowledgement from the client use 'SendMessage'
func (hub *Hub) UnsafeSendMessage(ID uuid.UUID, msg pkg.Serializable) error {
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
	clientMessage := &pkg.ClientMessage{
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
	timedCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Send message to client
	if err := hub.UnsafeSendMessage(ID, clientMessage); err != nil {
		return err
	}

	// Wait for ack from client
	select {
	case <-ack:
		// Success
		log.WithFields(log.Fields{
			"cid": cid,
			"id":  ID,
		}).Infoln("received ack")
		return nil
	case <-timedCtx.Done():
		log.WithFields(log.Fields{
			"cid": cid,
			"err": ctx.Err(),
		}).
			Warnln("error waiting for client to acknowledge message")
		return timedCtx.Err()
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
