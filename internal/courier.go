package internal

import (
	"context"
	"github.com/google/uuid"
)

type CourierServerImpl struct {
	UnimplementedCourierServer

	Hub *Hub
}

func (courier *CourierServerImpl) SendMessage(ctx context.Context, request *MessageRequest) (*MessageResponse, error) {
	id, err := uuid.Parse(request.Uuid)
	if err != nil {
		return nil, err
	}

	err = courier.Hub.SendMessage(ctx, id, request.Payload)
	if err != nil {
		return nil, err
	}

	return &MessageResponse{}, nil
}
