package internal

import (
	"database/sql"
	"github.com/david-wiles/groupme-clone/pkg"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"time"
)

type Message struct {
	roomID    uuid.UUID
	userID    uuid.UUID
	timestamp time.Time
	content   string
}

type Messages []Message

func (messages Messages) ToResponse() *pkg.MessageGetResponse {
	resp := &pkg.MessageGetResponse{}
	for _, message := range messages {
		resp.Messages = append(resp.Messages, pkg.MessageGetResponseMessage{
			UserID:    message.userID.String(),
			Content:   message.content,
			Timestamp: message.timestamp.Format(time.RFC3339),
		})
	}
	return resp
}

type MessageQueryEngine struct {
	*sql.DB
}

func (db MessageQueryEngine) CreateNewMessage(roomID, userID uuid.UUID, message string) error {
	id := uuid.New()
	stmt := `INSERT INTO "messages" ("id", "room", "content", "account_id") VALUES ($1, $2, $3, $4);`
	if _, err := db.Exec(stmt, id, roomID, message, userID); err != nil {
		log.
			WithFields(log.Fields{"err": err}).
			Errorln("unable to insert message into database")
		return err
	}

	return nil
}

func (db MessageQueryEngine) QueryMessages(roomID uuid.UUID, from, to time.Time) ([]Message, error) {
	stmt := `SELECT "room", "content", "account_id", "ts" FROM "messages" WHERE "room" = $1 AND "ts" > $2 AND "ts" < $3 ORDER BY ts DESC;`
	rows, err := db.Query(stmt, roomID, from, to)
	if err != nil {
		log.WithFields(log.Fields{"err": err, "room": roomID, "from": from, "to": to}).Errorln("unable to query messages")
		return nil, err
	}

	messages := []Message{}

	defer rows.Close()
	for rows.Next() {
		message := Message{}
		if err := rows.Scan(&message.roomID, &message.content, &message.userID, &message.timestamp); err != nil {
			log.WithFields(log.Fields{"err": err}).Warnln("unable to scan row")
		}
		messages = append(messages, message)
	}

	return messages, nil
}
