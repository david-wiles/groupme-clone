package internal

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"time"
)

type Message struct {
	ID        uuid.UUID `json:"id,omitempty"`
	RoomID    uuid.UUID `json:"roomId,omitempty"`
	UserID    uuid.UUID `json:"userId,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content,omitempty"`
}

func (message *Message) Encode() ([]byte, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(message); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type MessageQueryEngine struct {
	*sql.DB
}

func (db MessageQueryEngine) CreateNewMessage(roomID, userID uuid.UUID, ts time.Time, message string) error {
	id := uuid.New()
	stmt := `INSERT INTO "messages" ("id", "room", "content", "ts", "account_id") VALUES ($1, $2, $3, $4, $5);`
	if _, err := db.Exec(stmt, id, roomID, message, ts, userID); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("unable to insert message into database")
		return err
	}

	return nil
}

func (db MessageQueryEngine) QueryMessages(roomID uuid.UUID, from, to time.Time) ([]Message, error) {
	stmt := `SELECT "id", "room", "content", "account_id", "ts" FROM "messages" WHERE "room" = $1 AND "ts" > $2 AND "ts" < $3 ORDER BY ts DESC;`
	rows, err := db.Query(stmt, roomID, from, to)
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"room": roomID,
			"from": from,
			"to":   to,
		}).Errorln("unable to query messages")
		return nil, err
	}

	messages := []Message{}

	defer rows.Close()
	for rows.Next() {
		message := Message{}
		if err := rows.Scan(&message.ID, &message.RoomID, &message.Content, &message.UserID, &message.Timestamp); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Warnln("unable to scan row")
		}
		messages = append(messages, message)
	}

	return messages, nil
}
