package internal

import (
	"context"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"strings"
)

type CourierConns struct {
	cache map[string]CourierClient
	rdb   *redis.Client
}

func NewCourierConnCache(rdb *redis.Client) *CourierConns {
	return &CourierConns{
		cache: make(map[string]CourierClient),
		rdb:   rdb,
	}
}

func (conns *CourierConns) GetOrCreate(host string) (CourierClient, error) {
	if conn, ok := conns.cache[host]; ok {
		return conn, nil
	}

	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("unable to create GRPC connection")
		return nil, err
	}

	client := NewCourierClient(conn)
	conns.cache[host] = client

	return client, nil
}

func (conns *CourierConns) SendMessageTo(ctx context.Context, webhook string, message []byte) (bool, error) {
	splitWebhook := strings.Split(webhook, "/")
	if len(splitWebhook) != 2 {
		return false, nil
	}

	if conn, err := conns.GetOrCreate(splitWebhook[0]); err == nil {
		if _, err := conn.SendMessage(ctx, &MessageRequest{
			Uuid:    splitWebhook[1],
			Payload: message,
		}); err != nil {
			log.WithFields(log.Fields{
				"err":        err,
				"host":       splitWebhook[0],
				"connection": splitWebhook[1],
			}).Warnln("unable to send message")
			return true, err
		}
	}
	return true, nil
}

func (conns *CourierConns) BroadcastMessage(ctx context.Context, roomID uuid.UUID, message []byte) error {
	clients, err := conns.rdb.SMembers(ctx, roomID.String()).Result()
	if err != nil {
		return err
	}

	for _, client := range clients {
		if ok, _ := conns.SendMessageTo(ctx, client, message); !ok {
			if _, err := conns.rdb.SRem(ctx, roomID.String(), client).Result(); err != nil {
				log.WithFields(log.Fields{
					"err":    err,
					"client": client,
				}).Warnln("unable to remove client ID from cache")
			}
		}
	}

	return nil
}

func (conns *CourierConns) UnicastMessage(ctx context.Context, userID uuid.UUID, message []byte) error {
	client, err := conns.rdb.Get(ctx, userID.String()).Result()
	if err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"client": client,
		}).Errorln("unable to get client webhook")
		return err
	}

	ok, err := conns.SendMessageTo(ctx, client, message)
	if err != nil {
		if !ok {
			_ = conns.rdb.Del(ctx, userID.String())
		}

		return err
	}
	return nil
}
