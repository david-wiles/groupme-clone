package pkg

import "github.com/google/uuid"

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	ID    string `json:"id"`
}

type CreateAccountRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AccountResponse struct {
	ID       uuid.UUID `json:"id,omitempty"`
	Username string    `json:"username,omitempty"`
	Email    string    `json:"email,omitempty"`
}

type MessagePostRequest struct {
	RoomID  string `json:"roomId"`
	Message string `json:"message"`
}

type ClientRegisterRequest struct {
	ClientURL string `json:"clientUrl"`
}

type ClientUnRegisterRequest struct {
	ClientURL string `json:"clientUrl"`
}

type CreateRoomRequest struct {
	Name string `json:"name"`
	IsDm bool   `json:"isDm"`

	// Recipient is only used for requests to create a direct message room
	Recipient string `json:"recipient,omitempty"`
}

type ListRoomsResponse struct {
	Rooms []RoomResponse `json:"rooms"`
}

type RoomResponse struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

type MessageGetResponse struct {
	Messages []MessageGetResponseMessage `json:"messages"`
}

type MessageGetResponseMessage struct {
	UserID    string `json:"userId"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}
