package main

import (
	"encoding/json"
	"github.com/david-wiles/groupme-clone/internal"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type MessagePostRequest struct {
	RoomID  string `json:"roomId"`
	Message string `json:"message"`
}

func HandleMessagePost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()

	req := &MessagePostRequest{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(req); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warnln("unable to decode request body")
		w.WriteHeader(400)
		return
	}

	claims, ok := GetClaimsFromRequest(r)
	if !ok {
		log.WithFields(log.Fields{}).Errorln("unable to read claims from request")
		w.WriteHeader(500)
		return
	}

	roomID, err := uuid.Parse(req.RoomID)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	userID, err := uuid.Parse(claims.ID)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	members, err := roomQueryEngine.ListRoomMembers(roomID)
	if err != nil {
		if err == internal.NoMatchingRoomError {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	now := time.Now()
	id, err := messageQueryEngine.CreateNewMessage(roomID, userID, now, req.Message)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	// Create possibly encrypted message payload
	encoded, err := (&internal.Message{
		ID:        id,
		RoomID:    roomID,
		UserID:    userID,
		Timestamp: now,
		Content:   req.Message,
	}).Encode()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		})
		w.WriteHeader(500)
		return
	}

	if err := courierConns.BroadcastMessage(r.Context(), internal.FilterUUID(members, userID), encoded); err != nil {
		log.WithFields(log.Fields{
			"err":       err,
			"messageID": id,
			"userID":    userID,
		}).Warnln("unable to broadcast message")
	}
	_, _ = w.Write(encoded)
}

type ListMessagesResponse struct {
	Messages []internal.Message `json:"messages"`
}

func HandleMessageGet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Get required query parameters from and to
	fromRaw := r.URL.Query().Get("from")
	toRaw := r.URL.Query().Get("to")
	roomRaw := r.URL.Query().Get("room")

	if fromRaw == "" {
		w.WriteHeader(400)
		return
	}

	if roomRaw == "" {
		w.WriteHeader(400)
		return
	}

	from, err := time.Parse(time.RFC3339Nano, fromRaw)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	to, err := time.Parse(time.RFC3339Nano, toRaw)
	if err != nil {
		to = time.Now()
	}

	room, err := uuid.Parse(roomRaw)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	messages, err := messageQueryEngine.QueryMessages(room, from, to)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	internal.SerializeResponse(w, &ListMessagesResponse{messages})
}

func AddMessageRoutes(prefix string, router *httprouter.Router) {
	router.POST(prefix+"/message", JWTGuard(HandleMessagePost))
	router.GET(prefix+"/message", JWTGuard(HandleMessageGet))
}
