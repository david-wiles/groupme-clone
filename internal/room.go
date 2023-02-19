package internal

import (
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type Room struct {
	ID      uuid.UUID `json:"id,omitempty"`
	Name    string    `json:"name,omitempty"`
	Members []string  `json:"members,omitempty"`
}

type RoomQueryEngine struct {
	*sql.DB
}

var (
	NoMatchingRoomError = errors.New("no matching room found")
	AlreadyJoinedError  = errors.New("user is already a member of room")
)

func (db RoomQueryEngine) CreateRoom(name string, userID uuid.UUID) (*Room, error) {
	roomID := uuid.New()
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	stmt := `INSERT INTO "rooms" ("id", "name", "members") VALUES ($1, $2, $3);`
	if _, err := db.Exec(stmt, roomID, name, pq.StringArray{userID.String()}); err != nil {
		log.WithFields(log.Fields{"err": err}).Errorln("unable to execute query")
		if err := tx.Rollback(); err != nil {
			log.WithFields(log.Fields{"err": err}).Errorln("unable to rollback transaction")
		}
		return nil, err
	}

	stmt = `INSERT INTO "joined_rooms" ("account_id", "room_id", "is_admin") VALUES ($1, $2, true);`
	if _, err := db.Exec(stmt, userID, roomID); err != nil {
		log.WithFields(log.Fields{"err": err}).Errorln("unable to execute query")
		if err := tx.Rollback(); err != nil {
			log.WithFields(log.Fields{"err": err}).Errorln("unable to rollback transaction")
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		log.WithFields(log.Fields{"err": err}).Errorln("unable to commit transaction")
		return nil, err
	}

	return &Room{
		ID:      roomID,
		Name:    name,
		Members: []string{userID.String()},
	}, nil
}

func (db RoomQueryEngine) CreateDirectMessageRoom(name string, creatorID, recipientID uuid.UUID) (*Room, error) {
	roomID := uuid.New()

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	stmt := `INSERT INTO "rooms" ("id", "name", "members") VALUES ($1, $2, $3);`
	if _, err := db.Exec(stmt, roomID, name, pq.StringArray{creatorID.String(), recipientID.String()}); err != nil {
		log.WithFields(log.Fields{"err": err}).Errorln("unable to execute query")
		if err := tx.Rollback(); err != nil {
			log.WithFields(log.Fields{"err": err}).Errorln("unable to rollback transaction")
		}
		return nil, err
	}

	stmt = `INSERT INTO "joined_rooms" ("account_id", "room_id", "is_admin") VALUES ($1, $2, true);`
	if _, err := db.Exec(stmt, creatorID, roomID); err != nil {
		log.WithFields(log.Fields{"err": err}).Errorln("unable to execute query")
		if err := tx.Rollback(); err != nil {
			log.WithFields(log.Fields{"err": err}).Errorln("unable to rollback transaction")
		}
		return nil, err
	}

	stmt = `INSERT INTO "joined_rooms" ("account_id", "room_id", "is_admin") VALUES ($1, $2, true);`
	if _, err := db.Exec(stmt, recipientID, roomID); err != nil {
		log.WithFields(log.Fields{"err": err}).Errorln("unable to execute query")
		if err := tx.Rollback(); err != nil {
			log.WithFields(log.Fields{"err": err}).Errorln("unable to rollback transaction")
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		log.WithFields(log.Fields{"err": err}).Errorln("unable to commit transaction")
		return nil, err
	}

	return &Room{
		ID:      roomID,
		Name:    name,
		Members: []string{creatorID.String(), recipientID.String()},
	}, nil
}

func (db RoomQueryEngine) GetRoomByID(id uuid.UUID) (*Room, error) {
	stmt := `SELECT "id", "name", "members" FROM "rooms" WHERE "id" = $1;`
	if row := db.QueryRow(stmt, id.String()); row != nil {
		var (
			id      string
			name    string
			members pq.StringArray
		)

		if err := row.Scan(&id, &name, &members); err != nil {
			if err != sql.ErrNoRows {
				log.
					WithFields(log.Fields{"err": err}).
					Errorln("unable to scan account row")
				return nil, err
			} else {
				return nil, NoMatchingRoomError
			}
		}

		parsedID, err := uuid.Parse(id)
		if err != nil {
			log.
				WithFields(log.Fields{"err": err}).
				Errorln("unable to parse uuid")
			return nil, err
		}

		return &Room{
			ID:      parsedID,
			Name:    name,
			Members: members,
		}, nil
	}

	return nil, NoMatchingRoomError
}

func (db RoomQueryEngine) JoinRoom(roomID, userID uuid.UUID) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt := `INSERT INTO "joined_rooms" ("account_id", "room_id") VALUES ($1, $2);`
	if _, err := db.Exec(stmt, userID, roomID, true); err != nil {
		log.WithFields(log.Fields{"err": err}).Errorln("unable to execute query")
		if err := tx.Rollback(); err != nil {
			log.WithFields(log.Fields{"err": err}).Errorln("unable to rollback transaction")
		}
		return err
	}

	stmt = `UPDATE "rooms" SET "members" = array_append("members", $1) WHERE "id" = $2;`
	if _, err := db.Exec(stmt, userID, roomID, true); err != nil {
		log.WithFields(log.Fields{"err": err}).Errorln("unable to execute query")
		if err := tx.Rollback(); err != nil {
			log.WithFields(log.Fields{"err": err}).Errorln("unable to rollback transaction")
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		log.WithFields(log.Fields{"err": err}).Errorln("unable to commit transaction")
		return err
	}

	return nil
}

func (db RoomQueryEngine) AddAdmin(roomID, userID uuid.UUID) error {
	// Create joined_rooms entry for this user/room combination
	stmt := `UPDATE "joined_rooms" SET "is_admin" = TRUE WHERE "account_id" = $1 and "room_id" = $2;`
	if _, err := db.Exec(stmt, userID, roomID); err != nil {
		log.WithFields(log.Fields{"err": err}).Errorln("unable to create admin")
		return err
	}

	return nil
}

func (db RoomQueryEngine) AllJoinedRooms(userID uuid.UUID) ([]uuid.UUID, error) {
	stmt := `SELECT "room_id" FROM "joined_rooms" WHERE "account_id" = $1;`
	rows, err := db.Query(stmt, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NoMatchingRoomError
		}
		log.WithFields(log.Fields{"err": err}).Errorln("unable to query joined rooms")
		return nil, err
	}

	var roomID uuid.UUID
	var roomIDs []uuid.UUID

	for rows.Next() {
		if err := rows.Scan(&roomID); err != nil {
			log.WithFields(log.Fields{"err": err}).Errorln("unable to scan row")
			return nil, err
		}
		roomIDs = append(roomIDs, roomID)
	}

	return roomIDs, nil
}