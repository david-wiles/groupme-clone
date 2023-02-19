package main

import (
	"context"
	"github.com/david-wiles/groupme-clone/internal"
	"github.com/golang-jwt/jwt"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// JWTGuard is a middleware that ensures that the request has a valid JWT before passing to the next handler.
// This will return 403 without calling `next` if the JWT is invalid.
func JWTGuard(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		header := r.Header.Get("Authorization")
		var auth string

		// Instead of splitting the string, we will just remove the "Bearer " prefix
		if len(header) > 7 {
			auth = header[7:]
		} else {
			w.WriteHeader(400)
			return
		}

		token, err := internal.VerifyJWT(auth, jwtSecret)
		if err != nil {
			w.WriteHeader(500)
			log.
				WithFields(log.Fields{"err": err}).
				Warnln("unable to decode jwt")
			return
		}

		if !token.Valid {
			w.WriteHeader(403)
			return
		}

		// Put the parsed token into the request context and pass to the next handler
		ctx := context.WithValue(r.Context(), "jwt", token)
		req := r.WithContext(ctx)
		next(w, req, p)
	}
}

// GetClaimsFromRequest takes a *http.Request and parses the token to get the ClaimData in a usable form
func GetClaimsFromRequest(r *http.Request) (claimData internal.ClaimData, ok bool) {
	val := r.Context().Value("jwt")

	if token, ok := val.(*jwt.Token); ok {
		claims := token.Claims.(jwt.MapClaims)

		allOk := true

		if id, ok := claims["id"].(string); ok {
			claimData.ID = id
		} else {
			allOk = false
		}
		if username, ok := claims["username"].(string); ok {
			claimData.Username = username
		} else {
			allOk = false
		}
		if email, ok := claims["email"].(string); ok {
			claimData.Email = email
		} else {
			allOk = false
		}

		return claimData, allOk
	}

	return
}
