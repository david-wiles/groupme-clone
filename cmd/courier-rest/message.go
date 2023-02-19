package main

import (
	"context"
	"encoding/json"
	"github.com/david-wiles/groupme-clone/internal"
	"github.com/david-wiles/groupme-clone/pkg"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"strings"
)

type GRPCConns struct {
	cache map[string]internal.CourierClient
}

func (conns *GRPCConns) GetOrCreate(host string) (internal.CourierClient, error) {
	if conn, ok := conns.cache[host]; ok {
		return conn, nil
	}

	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Errorln("unable to create GRPC connection")
		return nil, err
	}

	client := internal.NewCourierClient(conn)
	conns.cache[host] = client

	return client, nil
}

var cachedGRPCConns = &GRPCConns{make(map[string]internal.CourierClient)}

func sendMessageTo(ctx context.Context, webhook, message string) (bool, error) {
	splitWebhook := strings.Split(webhook, "/")
	if len(splitWebhook) != 2 {
		return false, nil
	}

	if conn, err := cachedGRPCConns.GetOrCreate(splitWebhook[0]); err == nil {
		if _, err := conn.SendMessage(ctx, &internal.MessageRequest{
			Uuid:    splitWebhook[1],
			Payload: []byte(message),
		}); err != nil {
			log.
				WithFields(log.Fields{"err": err, "host": splitWebhook[0], "connection": splitWebhook[1]}).
				Warnln("unable to send message")
			return true, err
		}
	}
	return true, nil
}

func BroadcastMessage(ctx context.Context, roomID uuid.UUID, message string) error {
	clients, err := rdb.SMembers(ctx, roomID.String()).Result()
	if err != nil {
		return err
	}

	for _, client := range clients {
		if ok, _ := sendMessageTo(ctx, client, message); !ok {
			if _, err := rdb.SRem(ctx, roomID.String(), client).Result(); err != nil {
				log.WithFields(log.Fields{"err": err, "client": client}).Warnln("unable to remove client ID from cache")
			}
		}
	}

	return nil
}

func UnicastMessage(ctx context.Context, userID uuid.UUID, message string) error {
	client, err := rdb.Get(ctx, userID.String()).Result()
	if err != nil {
		log.WithFields(log.Fields{"err": err, "client": client}).Errorln("unable to get client webhook")
		return err
	}

	ok, err := sendMessageTo(ctx, client, message)
	if err != nil {
		if !ok {
			_ = rdb.Del(ctx, userID.String())
		}

		return err
	}
	return nil
}

func HandleMessagePost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()

	req := &pkg.MessagePostRequest{}
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

	room, err := roomQueryEngine.GetRoomByID(roomID)
	if err != nil {
		if err == internal.NoMatchingRoomError {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	if err := messageQueryEngine.CreateNewMessage(roomID, userID, req.Message); err != nil {
		w.WriteHeader(500)
		return
	}

	// Special case for rooms with only 2 members, i.e. a DM
	if len(room.Members) == 2 {
		for _, recipient := range room.Members {
			if recipient != userID.String() {
				recipientID, err := uuid.Parse(recipient)
				if err != nil {
					w.WriteHeader(201)
					return
				}
				if err := UnicastMessage(r.Context(), recipientID, req.Message); err != nil {
					w.WriteHeader(201)
					return
				}
			}
		}
	} else {
		if err := BroadcastMessage(r.Context(), roomID, req.Message); err != nil {
			w.WriteHeader(201)
			return
		}
	}
}

func AddMessageRoutes(router *httprouter.Router) {
	router.POST("/message", JWTGuard(HandleMessagePost))
}
