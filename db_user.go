package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type user struct {
	Email    string `json:"email"`
	password string
	ID       int `json:"id"`
}

func (u user) fullMarshal() map[string]any {
	return map[string]any{
		"id":       u.ID,
		"email":    u.Email,
		"password": u.password,
	}
}

func (db *fakeDB) addUser(w http.ResponseWriter, r *http.Request) {
	db.mut.Lock()
	defer db.mut.Unlock()
	var in struct {
		Password string
		Email    string
	}
	err := json.NewDecoder(r.Body).Decode(&in)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Error reading body")
		return
	}
	_, exist := db.Users[in.Email]
	if exist {
		sendError(w, http.StatusFound, "User already exists with that email")
		return
	}
	psw, err := bcrypt.GenerateFromPassword([]byte(in.Password), 0)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Error hashing password")
		return
	}
	id := len(db.Users) + 1
	db.Users[in.Email] = user{
		ID:       id,
		Email:    in.Email,
		password: string(psw),
	}
	db.sync()
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(db.Users[in.Email])
}

func (db *fakeDB) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string
		Email    string
		Expiry   int `json:"expires_in_seconds"`
	}
	err := json.NewDecoder(r.Body).Decode(&in)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Error reading body")
		return
	}
	usr, ok := db.Users[in.Email]
	if !ok {
		sendError(w, http.StatusUnauthorized, "User with given email does not exist")
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(usr.password), []byte(in.Password))
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Password does not match")
		return
	}
	if in.Expiry == 0 || in.Expiry > 24*60*60 {
		in.Expiry = 24 * 60 * 60
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(in.Expiry) * time.Second)),
		Subject:   strconv.Itoa(usr.ID),
	}).SignedString(db.jwt_secret)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Issue creating token")
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"id":    usr.ID,
		"email": usr.Email,
		"token": token,
	})
}

func (db *fakeDB) updateUser(w http.ResponseWriter, r *http.Request) {
	reqToken := r.Header.Get("Authorization")
	if reqToken == "" || !strings.HasPrefix(reqToken, "Bearer ") {
		sendError(w, http.StatusBadRequest, "Please provide a valid token")
		return
	}
	reqToken = strings.TrimPrefix(reqToken, "Bearer ")
	token, err := jwt.ParseWithClaims(reqToken, &jwt.RegisteredClaims{}, func(*jwt.Token) (interface{}, error) {
		return []byte(db.jwt_secret), nil
	})
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Token is not authorized")
		return
	}
	subj, _ := token.Claims.GetSubject()
	usrID, _ := strconv.Atoi(subj)
	var usr user
	for i := range db.Users {
		if db.Users[i].ID == usrID {
			usr = db.Users[i]
			break
		}
	}
	if usr.ID == 0 {
		sendError(w, http.StatusUnauthorized, "User ID not found")
		return
	}
	var bod struct {
		Email    string
		Password string
	}
	err = json.NewDecoder(r.Body).Decode(&bod)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Error decoding request body")
		return
	}
	if _, ok := db.Users[bod.Email]; usr.Email != bod.Email && !ok {
		sendError(w, http.StatusConflict, "Updated email is already in use")
		return
	}
	pwd, err := bcrypt.GenerateFromPassword([]byte(bod.Password), 0)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Error hashing password")
		return
	}
	db.mut.Lock()
	defer db.mut.Unlock()
	delete(db.Users, usr.Email)
	usr.Email = bod.Email
	usr.password = string(pwd)
}
