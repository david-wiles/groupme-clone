package internal

import (
	"database/sql"
	"errors"
	"github.com/david-wiles/groupme-clone/pkg"
	"github.com/google/uuid"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type Room struct {
	ID      uuid.UUID `json:"id,omitempty"`
	Name    string    `json:"name,omitempty"`
	Members []string  `json:"members,omitempty"`
}

type Rooms []Room

func (rooms Rooms) ToResponse() *pkg.ListRoomsResponse {
	resp := &pkg.ListRoomsResponse{
		Rooms: []pkg.RoomResponse{},
	}
	for _, room := range rooms {
		resp.Rooms = append(resp.Rooms, pkg.RoomResponse{
			ID:      room.ID.String(),
			Name:    room.Name,
			Members: room.Members,
		})
	}
	return resp
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

	stmt := `INSERT INTO "joined_rooms" ("account_id", "room_id", "is_admin") VALUES ($1, $2, false);`
	if _, err := db.Exec(stmt, userID, roomID); err != nil {
		log.WithFields(log.Fields{"err": err}).Errorln("unable to execute query")
		if err := tx.Rollback(); err != nil {
			log.WithFields(log.Fields{"err": err}).Errorln("unable to rollback transaction")
		}
		return err
	}

	stmt = `UPDATE "rooms" SET "members" = array_append("members", $1) WHERE "id" = $2;`
	if _, err := db.Exec(stmt, userID, roomID); err != nil {
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

func (db RoomQueryEngine) ListJoinedRooms(userID uuid.UUID) ([]uuid.UUID, error) {
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

func (db RoomQueryEngine) GetJoinedRooms(userID uuid.UUID) ([]Room, error) {
	stmt := `SELECT id, name, members FROM "rooms" LEFT JOIN joined_rooms jr on rooms.id = jr.room_id WHERE jr.account_id = $1;`
	rows, err := db.Query(stmt, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []Room{}, nil
		}
	}

	rooms := []Room{}

	for rows.Next() {
		room := Room{}
		members := pq.StringArray{}
		if err := rows.Scan(&room.ID, &room.Name, &members); err != nil {
			log.WithFields(log.Fields{"err": err}).Warnln("unable to scan row")
		}
		room.Members = members
		rooms = append(rooms, room)
	}

	return rooms, nil
}
