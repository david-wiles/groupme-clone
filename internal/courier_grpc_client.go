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
	cache  map[string]CourierClient
	client RegistrationEngine
}

func NewCourierConnCache(rdb *redis.Client) *CourierConns {
	return &CourierConns{
		cache:  make(map[string]CourierClient),
		client: RegistrationEngine{rdb},
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

func (conns *CourierConns) BroadcastMessage(ctx context.Context, users []uuid.UUID, message []byte) error {
	webhooks, err := conns.client.ListUsersWebhook(ctx, users)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"users": users,
		}).Errorln("unable to list webhooks")
		return err
	}

	for _, webhook := range webhooks {
		_, _ = conns.SendMessageTo(ctx, webhook, message)
	}
	return nil
}

func (conns *CourierConns) UnicastMessage(ctx context.Context, userID uuid.UUID, message []byte) error {
	webhook, err := conns.client.GetUserWebhook(ctx, userID)
	if err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"userID": userID,
		}).Errorln("unable to get client webhook")
		return err
	}

	ok, err := conns.SendMessageTo(ctx, webhook, message)
	if err != nil {
		if !ok {
			_ = conns.client.RemoveUserWebhook(ctx, userID)
		}

		return err
	}
	return nil
}
