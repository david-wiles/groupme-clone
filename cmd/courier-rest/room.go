package main

import (
	"encoding/json"
	"github.com/david-wiles/groupme-clone/internal"
	"github.com/david-wiles/groupme-clone/pkg"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func HandleCreateRoom(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()

	req := &pkg.CreateRoomRequest{}
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
		w.WriteHeader(500)
		return
	}

	var room *internal.Room
	if req.IsDm && req.Recipient != "" {
		recipientID, err := uuid.Parse(req.Recipient)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		room, err = roomQueryEngine.CreateDirectMessageRoom(req.Name, parsedUserID, recipientID)
		if err != nil {
			w.WriteHeader(500)
			return
		}
	} else {
		room, err = roomQueryEngine.CreateRoom(req.Name, parsedUserID)
		if err != nil {
			w.WriteHeader(500)
			return
		}
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(room); err != nil {
		log.
			WithFields(log.Fields{
				"room": room,
				"err":  err,
			}).
			Errorln("unable to encode room data")
		w.WriteHeader(500)
	}

	w.WriteHeader(201)
}

func HandleGetRoom(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")
	parsedID, err := uuid.Parse(id)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	room, err := roomQueryEngine.GetRoomByID(parsedID)
	if err != nil {
		if err == internal.NoMatchingRoomError {
			w.WriteHeader(404)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(room); err != nil {
		log.
			WithFields(log.Fields{
				"room": room,
				"err":  err,
			}).
			Errorln("unable to encode room data")
		w.WriteHeader(500)
	}
}

func HandleJoinRoom(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")
	parsedRoomID, err := uuid.Parse(id)
	if err != nil {
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
		w.WriteHeader(500)
		return
	}

	if err := roomQueryEngine.JoinRoom(parsedRoomID, parsedUserID); err != nil {
		if err == internal.AlreadyJoinedError {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(500)
		}
		return
	}
}

func AddRoomRoutes(router *httprouter.Router) {
	router.POST("/room/", JWTGuard(HandleCreateRoom))
	//router.PATCH("/room/:id", HandleUpdateRoom)
	router.GET("/room/:id", JWTGuard(HandleGetRoom))
	router.POST("/room/:id/join", JWTGuard(HandleJoinRoom))
}