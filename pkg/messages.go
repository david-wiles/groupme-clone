package pkg

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
)

type Serializable interface {
}

type ClientAck struct {
	Cid          string `json:"cid"`
	Serializable `json:"serializable,omitempty"`
}

type ClientMessage struct {
	Payload      []byte `json:"payload"`
	Cid          string `json:"cid,omitempty"`
	Acknowledge  bool   `json:"acknowledge,omitempty"`
	Serializable `json:"serializable,omitempty"`
}

type MessagePayload struct {
	RoomID    string `json:"roomId"`
	UserID    string `json:"userId"`
	Timestamp string `json:"timestamp"`
	Content   string `json:"content"`
}

type WhoAmIResponse struct {
	Webhook      string `json:"webhook"`
	Serializable `json:"serializable,omitempty"`
}

type Serializer interface {
	Serialize(Serializable) (int, []byte, error)
	Deserialize([]byte, Serializable) error
}

type JSONSerializer struct{}

func (JSONSerializer) Serialize(s Serializable) (t int, b []byte, err error) {
	buf := bytes.NewBuffer(b)
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(s); err != nil {
		return 0, nil, err
	}
	return websocket.TextMessage, buf.Bytes(), nil
}

func (JSONSerializer) Deserialize(b []byte, s Serializable) error {
	buf := bytes.NewReader(b)
	decoder := json.NewDecoder(buf)
	if err := decoder.Decode(s); err != nil {
		return err
	}
	return nil
}
