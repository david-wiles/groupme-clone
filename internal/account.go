package internal

import (
	"database/sql"
	"errors"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Account struct {
	ID       uuid.UUID `json:"id,omitempty"`
	Username string    `json:"username,omitempty"`
	Email    string    `json:"email,omitempty"`
}

type AccountQueryEngine struct {
	*sql.DB
}

var NoMatchingUserError = errors.New("no matching user account found")

// CreateAccount makes a new account, saves into the database, and returns a struct representing the object
func (db AccountQueryEngine) CreateAccount(username, email, pass string) (*Account, error) {
	id := uuid.New()
	stmt := `INSERT INTO "accounts" ("id", "username", "email", "hashed_pass") VALUES ($1, $2, $3, $4);`
	if _, err := db.Exec(stmt, id, username, email, pass); err != nil {
		log.WithFields(log.Fields{
			"err":      err,
			"username": username,
			"email":    email,
		}).Errorln("unable to insert account into database")
		return nil, err
	}

	return &Account{
		ID:       id,
		Username: username,
		Email:    email,
	}, nil
}

func (db AccountQueryEngine) GetAccountByUsername(username string) (*Account, error) {
	stmt := `SELECT "id", "username", "email", "hashed_pass" FROM "accounts" WHERE "username" = $1;`
	if row := db.QueryRow(stmt, username); row != nil {
		var (
			id         string
			username   string
			email      string
			hashedPass string
		)

		if err := row.Scan(&id, &username, &email, &hashedPass); err != nil {
			if err != sql.ErrNoRows {
				log.WithFields(log.Fields{
					"err": err,
				}).Errorln("unable to scan account row")
				return nil, err
			} else {
				return nil, NoMatchingUserError
			}
		}

		parsedID, err := uuid.Parse(id)
		if err != nil {
			log.WithFields(log.Fields{
				"err": err,
				"id":  id,
			}).Errorln("unable to parse uuid")
			return nil, err
		}

		return &Account{
			ID:       parsedID,
			Username: username,
			Email:    email,
		}, nil
	}

	return nil, NoMatchingUserError
}

func (db AccountQueryEngine) VerifyPassword(username, password string) (bool, error) {
	stmt := `SELECT "hashed_pass" FROM "accounts" WHERE "username" = $1;`
	if row := db.QueryRow(stmt, username); row != nil {
		var hashedPass string

		if err := row.Scan(&hashedPass); err != nil {
			if err != sql.ErrNoRows {
				log.WithFields(log.Fields{
					"err": err,
				}).Errorln("unable to scan account row")
				return false, err
			} else {
				return false, NoMatchingUserError
			}
		}

		return ComparePasswordWithHash(hashedPass, password)
	}

	return false, NoMatchingUserError
}
