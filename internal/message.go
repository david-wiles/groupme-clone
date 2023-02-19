package internal

import (
	"database/sql"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

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
