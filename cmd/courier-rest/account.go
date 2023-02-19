package main

import (
	"encoding/json"
	"github.com/david-wiles/groupme-clone/internal"
	"github.com/david-wiles/groupme-clone/pkg"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func HandleCreateAccount(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)

	req := &pkg.CreateAccountRequest{}
	if err := decoder.Decode(req); err != nil {
		log.
			WithFields(log.Fields{"err": err}).
			Warnln("cannot decode request")
		w.WriteHeader(400)
		return
	}

	pass, err := internal.HashPassword(req.Password)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	account, err := accountQueryEngine.CreateAccount(req.Username, req.Email, pass)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	token, err := internal.GenerateJWT(account, jwtSecret)
	if err != nil {
		log.
			WithFields(log.Fields{"err": err}).
			Warnln("unable to generate JWT")
		w.WriteHeader(500)
		return
	}

	resp := &pkg.LoginResponse{token}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(resp); err != nil {
		log.
			WithFields(log.Fields{"err": err}).
			Warnln("unable to write response")
		w.WriteHeader(500)
	}

	w.WriteHeader(201)
}

func HandleLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	req := &pkg.LoginRequest{}

	if err := decoder.Decode(req); err != nil {
		log.
			WithFields(log.Fields{"err": err}).
			Warnln("cannot decode request")
		w.WriteHeader(400)
		return
	}

	ok, err := accountQueryEngine.VerifyPassword(req.Username, req.Password)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	if !ok {
		w.WriteHeader(403)
		return
	}

	account, err := accountQueryEngine.GetAccountByUsername(req.Username)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	token, err := internal.GenerateJWT(account, jwtSecret)
	if err != nil {
		log.
			WithFields(log.Fields{"err": err}).
			Warnln("unable to generate JWT")
		w.WriteHeader(500)
		return
	}

	resp := &pkg.LoginResponse{token}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(resp); err != nil {
		log.
			WithFields(log.Fields{"err": err}).
			Warnln("unable to write response")
		w.WriteHeader(500)
	}
}

func HandleGetUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	username := p.ByName("username")

	account, err := accountQueryEngine.GetAccountByUsername(username)
	if err != nil {
		if err == internal.NoMatchingUserError {
			w.WriteHeader(404)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(account); err != nil {
		log.
			WithFields(log.Fields{"err": err}).
			Warnln("unable to write response")
		w.WriteHeader(500)
	}
}

func AddAccountRoutes(router *httprouter.Router) {
	router.POST("/account/", HandleCreateAccount)
	router.POST("/account/login", HandleLogin)
	//router.GET("/account/", HandleGetSelf)
	//router.PATCH("/account/me", HandleAccountUpdate)
	router.GET("/account/:username", JWTGuard(HandleGetUser))
}
