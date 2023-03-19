package main

import (
	"encoding/json"
	"github.com/david-wiles/groupme-clone/internal"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

type CreateAccountRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func HandleCreateAccount(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)

	req := &CreateAccountRequest{}
	if err := decoder.Decode(req); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warnln("cannot decode request")
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
		log.WithFields(log.Fields{
			"err": err,
		}).Warnln("unable to generate JWT")
		w.WriteHeader(500)
		return
	}

	internal.SerializeResponse(w, &LoginResponse{token, account.ID.String()})
}

type ListAccountResponse struct {
	Users []*internal.Account `json:"users"`
}

func HandleAccountList(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	usersRaw := r.URL.Query().Get("users")

	listRaw := strings.Split(usersRaw, ",")

	var users []uuid.UUID
	for _, raw := range listRaw {
		u, err := uuid.Parse(raw)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		users = append(users, u)
	}

	var accounts []*internal.Account
	for _, userID := range users {
		if account, err := accountQueryEngine.GetAccount(userID); err == nil {
			accounts = append(accounts, account)
		}
	}

	internal.SerializeResponse(w, &ListAccountResponse{accounts})
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	ID    string `json:"id"`
}

func HandleLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	req := &LoginRequest{}

	if err := decoder.Decode(req); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warnln("cannot decode request")
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
		log.WithFields(log.Fields{
			"err":       err,
			"accountID": account.ID,
		}).Warnln("unable to generate JWT")
		w.WriteHeader(500)
		return
	}

	internal.SerializeResponse(w, &LoginResponse{token, account.ID.String()})
}

func HandleGetUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")

	parsedId, err := uuid.Parse(id)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	account, err := accountQueryEngine.GetAccount(parsedId)
	if err != nil {
		if err == internal.NoMatchingUserError {
			w.WriteHeader(404)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	internal.SerializeResponse(w, account)
}

func AddAccountRoutes(prefix string, router *httprouter.Router) {
	router.POST(prefix+"/account", HandleCreateAccount)
	router.GET(prefix+"/account", JWTGuard(HandleAccountList))
	router.POST(prefix+"/account/login", HandleLogin)
	//router.GET("/account/", HandleGetSelf)
	//router.PATCH("/account/me", HandleAccountUpdate)
	router.GET(prefix+"/account/:id", JWTGuard(HandleGetUser))
}
