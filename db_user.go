package main

import (
	"encoding/json"
	"net/http"

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
	json.NewEncoder(w).Encode(usr)
}
