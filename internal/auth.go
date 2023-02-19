package internal

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

func HashPassword(pass string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 0)
	if err != nil {
		log.
			WithFields(log.Fields{"err": err}).
			Errorln("unable to hash password")
		return "", err
	}
	return string(hash), nil
}

func ComparePasswordWithHash(hash, pass string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
	if err != nil {
		if err != bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type ClaimData struct {
	ID       string
	Username string
	Email    string
}

func GenerateJWT(account *Account, key interface{}) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["id"] = account.ID
	claims["username"] = account.Username
	claims["email"] = account.Email
	return token.SignedString(key)
}

func VerifyJWT(auth string, key interface{}) (*jwt.Token, error) {
	return jwt.Parse(auth, func(token *jwt.Token) (interface{}, error) { return key, nil })
}

func GetAndVerifyJWT(jwtSecret interface{}, r *http.Request) (*http.Request, error) {
	header := r.Header.Get("Authorization")
	var auth string

	// Instead of splitting the string, we will just remove the "Bearer " prefix
	if len(header) > 7 {
		auth = header[7:]
	} else {
		return nil, errors.New("invalid authorization header")
	}

	token, err := VerifyJWT(auth, jwtSecret)
	if err != nil {
		return nil, errors.New("unable to decode token")
	}

	if !token.Valid {
		return nil, errors.New("token is invalid")
	}

	// Put the parsed token into the request context to pass to the next handler
	ctx := context.WithValue(r.Context(), "jwt", token)
	return r.WithContext(ctx), nil
}
