package main

import (
	"github.com/david-wiles/groupme-clone/internal"
	"github.com/golang-jwt/jwt"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

// JWTGuard is a middleware that ensures that the request has a valid JWT before passing to the next handler.
// This will return 403 without calling `next` if the JWT is invalid.
func JWTGuard(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		req, err := internal.VerifyJWTFromRequest(jwtSecret, r)
		if err != nil {
			w.WriteHeader(403)
			return
		}

		next(w, req, p)
	}
}

// GetClaimsFromRequest takes a *http.Request and parses the token to get the ClaimData in a usable form
func GetClaimsFromRequest(r *http.Request) (claimData internal.ClaimData, ok bool) {
	val := r.Context().Value("jwt")

	if token, ok := val.(*jwt.Token); ok {
		return internal.GetClaimsFromToken(token)
	}

	return
}
