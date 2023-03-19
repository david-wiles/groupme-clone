package internal

import (
	"context"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"time"
)

type RegistrationEngine struct {
	*redis.Client
}

func (rdb RegistrationEngine) SetUserWebhook(ctx context.Context, webhook string, userID uuid.UUID) error {
	if _, err := rdb.Set(ctx, userID.String(), webhook, time.Hour*168).Result(); err != nil {
		return err
	}
	return nil
}

func (rdb RegistrationEngine) GetUserWebhook(ctx context.Context, userID uuid.UUID) (string, error) {
	w, err := rdb.Get(ctx, userID.String()).Result()
	if err != nil {
		return "", err
	}
	return w, nil
}

func (rdb RegistrationEngine) ListUsersWebhook(ctx context.Context, users []uuid.UUID) ([]string, error) {
	var usersList []string
	for _, user := range users {
		usersList = append(usersList, user.String())
	}

	list, err := rdb.MGet(ctx, usersList...).Result()
	if err != nil {
		return nil, err
	}

	var webhooks []string
	for _, v := range list {
		if webhook, ok := v.(string); ok {
			webhooks = append(webhooks, webhook)
		}
	}

	return webhooks, nil
}

func (rdb RegistrationEngine) RemoveUserWebhook(ctx context.Context, userID uuid.UUID) error {
	if _, err := rdb.Del(ctx, userID.String(), userID.String()).Result(); err != nil {
		return err
	}
	return nil
}
