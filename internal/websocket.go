package internal

import (
	"github.com/david-wiles/groupme-clone/pkg"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// WebsocketConnection represents an active websocket that is registered with the courier's hub.
// The websocket is identified with a UUID and can be written to using a channel of type
// WebsocketMessage.
type WebsocketConnection struct {
	// Writes should be used to send a message to the connection. This channel is consumed by
	// writeWorker, which reads messages from the channel until it is closed
	Writes chan pkg.Serializable
	pkg.Serializer

	// ID is a UUID identifying the websocket
	ID uuid.UUID

	// userID corresponds to the user listening to this websocket
	userID uuid.UUID

	// conn is the underlying websocket connection.
	conn *websocket.Conn

	// hub is a back reference to the Hub managing this websocket connection
	hub *Hub
}

// NewWebsocketConnection will create a new websocket and generate a UUID
func NewWebsocketConnection(conn *websocket.Conn, hub *Hub, userID uuid.UUID) *WebsocketConnection {
	ws := &WebsocketConnection{
		Writes:     make(chan pkg.Serializable, 64),
		Serializer: hub.Serializer,
		ID:         uuid.New(),
		userID:     userID,
		conn:       conn,
		hub:        hub,
	}

	go ws.readWorker()
	go ws.writeWorker()

	return ws
}

// readWorker continually reads messages from the websocket until closed
func (ws *WebsocketConnection) readWorker() {
	for {
		_, bytes, err := ws.conn.ReadMessage()
		if err != nil {
			// Close this connection, mark as closed and close channel
			ws.unregister()
			return
		}

		msg := &pkg.ClientAck{}
		if err := ws.Deserialize(bytes, msg); err != nil {
			log.WithFields(log.Fields{
				"id":  ws.ID,
				"err": err,
			}).Warnln("unable to decode client message")
		}

		if err := ws.hub.Acknowledge(msg.Cid); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Warnln("failed to acknowledge message")
		}
	}
}

// writeWorker continually writes messages from the Writes channel until closed
func (ws *WebsocketConnection) writeWorker() {
	for msg := range ws.Writes {
		t, b, err := ws.Serialize(msg)
		if err != nil {
			log.WithFields(log.Fields{
				"err": err,
				"id":  ws.ID,
			}).Warnln("unable to serialize message er")
			continue
		}

		if err := ws.conn.WriteMessage(t, b); err != nil {
			log.WithFields(log.Fields{
				"err": err,
				"id":  ws.ID,
			}).Warnln("error writing message")
		}
	}
}

// unregister removes itself from the hub and closes the underlying connection
func (ws *WebsocketConnection) unregister() {
	log.WithFields(log.Fields{
		"id": ws.ID,
	}).Infoln("removing websocket")

	// Remove the connection from the hub first to prevent other goroutines from writing to
	// the websocket while resources are being cleaned up
	ws.hub.UnregisterConnection(ws.ID)

	// Closing the connection will cause readWorker to exit
	_ = ws.conn.Close()

	// Closing the writes channel will cause writeWorker to exit
	close(ws.Writes)
}
