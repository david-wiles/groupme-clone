package internal

import (
	"database/sql"
	"errors"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Room struct {
	ID   uuid.UUID `json:"id,omitempty"`
	Name string    `json:"name,omitempty"`
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

	stmt := `INSERT INTO "rooms" ("id", "name") VALUES ($1, $2);`
	if _, err := db.Exec(stmt, roomID, name); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("unable create room")
		if err := tx.Rollback(); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Errorln("unable to rollback transaction")
		}
		return nil, err
	}

	stmt = `INSERT INTO "joined_rooms" ("account_id", "room_id", "is_admin") VALUES ($1, $2, true);`
	if _, err := db.Exec(stmt, userID, roomID); err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"roomID": roomID,
			"userID": userID,
		}).Errorln("unable to join room")
		if err := tx.Rollback(); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Errorln("unable to rollback transaction")
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("unable to commit transaction")
		return nil, err
	}

	return &Room{
		ID:   roomID,
		Name: name,
	}, nil
}

func (db RoomQueryEngine) CreateDirectMessageRoom(name string, creatorID, recipientID uuid.UUID) (*Room, error) {
	roomID := uuid.New()

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	stmt := `INSERT INTO "rooms" ("id", "name") VALUES ($1, $2);`
	if _, err := db.Exec(stmt, roomID, name); err != nil {
		log.WithFields(log.Fields{
			"err":         err,
			"creatorID":   creatorID,
			"recipientID": recipientID,
		}).Errorln("unable to create DM room")
		if err := tx.Rollback(); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Errorln("unable to rollback transaction")
		}
		return nil, err
	}

	stmt = `INSERT INTO "joined_rooms" ("account_id", "room_id", "is_admin") VALUES ($1, $2, true);`
	if _, err := db.Exec(stmt, creatorID, roomID); err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"roomID": roomID,
			"userID": creatorID,
		}).Errorln("unable to join room")
		if err := tx.Rollback(); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Errorln("unable to rollback transaction")
		}
		return nil, err
	}

	stmt = `INSERT INTO "joined_rooms" ("account_id", "room_id", "is_admin") VALUES ($1, $2, true);`
	if _, err := db.Exec(stmt, recipientID, roomID); err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"roomID": roomID,
			"userID": recipientID,
		}).Errorln("unable to join room")
		if err := tx.Rollback(); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Errorln("unable to rollback transaction")
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("unable to commit transaction")
		return nil, err
	}

	return &Room{
		ID:   roomID,
		Name: name,
	}, nil
}

func (db RoomQueryEngine) GetRoomByID(id uuid.UUID) (*Room, error) {
	stmt := `SELECT "id", "name" FROM "rooms" WHERE "id" = $1;`
	if row := db.QueryRow(stmt, id.String()); row != nil {
		var (
			id   string
			name string
		)

		if err := row.Scan(&id, &name); err != nil {
			if err != sql.ErrNoRows {
				log.WithFields(log.Fields{
					"err":    err,
					"roomID": id,
				}).Errorln("unable to scan room row")
				return nil, err
			} else {
				return nil, NoMatchingRoomError
			}
		}

		parsedID, err := uuid.Parse(id)
		if err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Errorln("unable to parse uuid")
			return nil, err
		}

		return &Room{
			ID:   parsedID,
			Name: name,
		}, nil
	}

	return nil, NoMatchingRoomError
}

func (db RoomQueryEngine) JoinRoom(roomID, userID uuid.UUID) error {
	stmt := `INSERT INTO "joined_rooms" ("account_id", "room_id", "is_admin") VALUES ($1, $2, false);`
	if _, err := db.Exec(stmt, userID, roomID); err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"roomID": roomID,
			"userID": userID,
		}).Errorln("unable to join room")
		return err
	}

	return nil
}

func (db RoomQueryEngine) ListRoomMembers(roomID uuid.UUID) ([]uuid.UUID, error) {
	stmt := `SELECT "account_id" FROM "joined_rooms" WHERE "room_id" = $1;`
	rows, err := db.Query(stmt, roomID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NoMatchingRoomError
		}
		log.WithFields(log.Fields{
			"err":    err,
			"roomID": roomID,
		}).Errorln("unable to query joined rooms")
	}

	var userID uuid.UUID
	var users []uuid.UUID

	for rows.Next() {
		if err := rows.Scan(&userID); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Errorln("unable to scan row")
			return nil, err
		}
		users = append(users, userID)
	}

	return users, nil
}

func (db RoomQueryEngine) AddAdmin(roomID, userID uuid.UUID) error {
	// Create joined_rooms entry for this user/room combination
	stmt := `UPDATE "joined_rooms" SET "is_admin" = TRUE WHERE "account_id" = $1 and "room_id" = $2;`
	if _, err := db.Exec(stmt, userID, roomID); err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"userID": userID,
			"roomID": roomID,
		}).Errorln("unable to create admin")
		return err
	}

	return nil
}

func (db RoomQueryEngine) ListJoinedRooms(userID uuid.UUID) ([]uuid.UUID, error) {
	stmt := `SELECT "room_id" FROM "joined_rooms" WHERE "account_id" = $1;`
	rows, err := db.Query(stmt, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NoMatchingRoomError
		}
		log.WithFields(log.Fields{
			"err":    err,
			"userID": userID,
		}).Errorln("unable to query joined rooms")
		return nil, err
	}

	var roomID uuid.UUID
	var roomIDs []uuid.UUID

	for rows.Next() {
		if err := rows.Scan(&roomID); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Errorln("unable to scan row")
			return nil, err
		}
		roomIDs = append(roomIDs, roomID)
	}

	return roomIDs, nil
}

func (db RoomQueryEngine) GetJoinedRooms(userID uuid.UUID) ([]Room, error) {
	stmt := `SELECT id, name FROM "rooms" LEFT JOIN joined_rooms jr on rooms.id = jr.room_id WHERE jr.account_id = $1;`
	rows, err := db.Query(stmt, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []Room{}, nil
		}
		log.WithFields(log.Fields{
			"err":    err,
			"userID": userID,
		}).Errorln("unable to query joined rooms")
	}

	rooms := []Room{}

	for rows.Next() {
		room := Room{}
		if err := rows.Scan(&room.ID, &room.Name); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Warnln("unable to scan row")
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}
