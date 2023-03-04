package main

import (
	"context"
	"encoding/json"
	"github.com/david-wiles/groupme-clone/internal"
	"github.com/david-wiles/groupme-clone/pkg"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func AddClientToRoom(ctx context.Context, clientURL string, roomID uuid.UUID) error {
	if _, err := rdb.SAdd(ctx, roomID.String(), clientURL).Result(); err != nil {
		return err
	}
	return nil
}

func RemoveClientFromRoom(ctx context.Context, clientURL string, roomID uuid.UUID) error {
	if _, err := rdb.SRem(ctx, roomID.String(), clientURL).Result(); err != nil {
		return err
	}
	return nil
}

func MapUserToClient(ctx context.Context, clientURL string, userID uuid.UUID) error {
	if _, err := rdb.Set(ctx, userID.String(), clientURL, time.Hour*168).Result(); err != nil {
		return err
	}
	return nil
}

func HandleSelfRegister(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()

	req := &pkg.ClientRegisterRequest{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(req); err != nil {
		log.WithFields(log.Fields{"err": err}).Warnln("unable to decode request body")
		w.WriteHeader(400)
		return
	}

	claims, ok := GetClaimsFromRequest(r)
	if !ok {
		log.WithFields(log.Fields{}).Errorln("unable to read claims from request")
		w.WriteHeader(500)
		return
	}

	parsedUserID, err := uuid.Parse(claims.ID)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	// Find rooms that the user has joined
	roomIDs, err := roomQueryEngine.ListJoinedRooms(parsedUserID)
	if err != nil && err != internal.NoMatchingRoomError {
		w.WriteHeader(500)
		return
	}

	// Map the user's ID to their client URL
	if err := MapUserToClient(r.Context(), req.ClientURL, parsedUserID); err != nil {
		w.WriteHeader(500)
		return
	}

	// Add the user to list of clients for each room in Redis
	for _, roomID := range roomIDs {
		if err := AddClientToRoom(r.Context(), req.ClientURL, roomID); err != nil {
			log.
				WithFields(log.Fields{"err": err, "roomID": roomID}).
				Errorln("unable to add client to redis")
		}
	}
}

func HandleSelfUnregister(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()

	req := &pkg.ClientUnRegisterRequest{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(req); err != nil {
		log.WithFields(log.Fields{"err": err}).Warnln("unable to decode request body")
		w.WriteHeader(400)
		return
	}

	claims, ok := GetClaimsFromRequest(r)
	if !ok {
		log.WithFields(log.Fields{}).Errorln("unable to read claims from request")
		w.WriteHeader(500)
		return
	}

	parsedUserID, err := uuid.Parse(claims.ID)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	// Find rooms the user is a member of
	roomIDs, err := roomQueryEngine.ListJoinedRooms(parsedUserID)
	if err != nil && err != internal.NoMatchingRoomError {
		w.WriteHeader(500)
		return
	}

	// Remove the user from list of connected clients for each room
	for _, roomID := range roomIDs {
		if err := RemoveClientFromRoom(r.Context(), req.ClientURL, roomID); err != nil {
			log.
				WithFields(log.Fields{"err": err, "roomID": roomID}).
				Errorln("unable to remove client from redis")
		}
	}
}

func AddWebsocketRoutes(router *httprouter.Router) {
	router.POST("/client/register", JWTGuard(HandleSelfRegister))
	router.POST("/client/unregister", JWTGuard(HandleSelfUnregister))
}
