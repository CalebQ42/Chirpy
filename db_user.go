package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type user struct {
	Email    string `json:"email"`
	password string
	ID       int  `json:"id"`
	Red      bool `json:"is_chirpy_red"`
}

func (u user) fullMarshal() map[string]any {
	return map[string]any{
		"id":            u.ID,
		"email":         u.Email,
		"password":      u.password,
		"is_chirpy_red": u.Red,
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
	if in.Password == "" || in.Email == "" {
		sendError(w, http.StatusBadRequest, "Both email and pasword is required")
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
	db.Users[in.Email] = &user{
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
	if in.Password == "" || in.Email == "" {
		sendError(w, http.StatusBadRequest, "Both email and pasword is required")
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
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		Subject:   strconv.Itoa(usr.ID),
	}).SignedString([]byte(db.jwt_secret))
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Issue creating token")
		fmt.Println(err)
		return
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy-refresh",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(60 * 24 * time.Hour)),
		Subject:   strconv.Itoa(usr.ID),
	}).SignedString([]byte(db.jwt_secret))
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Issue creating token")
		fmt.Println(err)
		return
	}
	err = json.NewEncoder(w).Encode(map[string]any{
		"id":            usr.ID,
		"email":         usr.Email,
		"is_chirpy_red": usr.Red,
		"token":         accessToken,
		"refresh_token": refreshToken,
	})
	if err != nil {
		fmt.Println(err)
	}
}

func (db *fakeDB) updateUser(w http.ResponseWriter, r *http.Request) {
	reqToken := r.Header.Get("Authorization")
	if reqToken == "" || !strings.HasPrefix(reqToken, "Bearer ") {
		sendError(w, http.StatusUnauthorized, "Please provide a valid token")
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
	issuer, err := token.Claims.GetIssuer()
	if err != nil || issuer != "chirpy-access" {
		sendError(w, http.StatusUnauthorized, "Please provide an access token")
		return
	}
	subj, _ := token.Claims.GetSubject()
	usrID, _ := strconv.Atoi(subj)
	var usr *user
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
	db.mut.Lock()
	defer db.mut.Unlock()
	if _, ok := db.Users[bod.Email]; usr.Email != bod.Email && ok {
		sendError(w, http.StatusConflict, "Updated email is already in use")
		return
	}
	pwd, err := bcrypt.GenerateFromPassword([]byte(bod.Password), 0)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Error hashing password")
		return
	}
	delete(db.Users, usr.Email)
	usr.Email = bod.Email
	usr.password = string(pwd)
	db.Users[usr.Email] = usr
	json.NewEncoder(w).Encode(usr)
}

func (db *fakeDB) refresh(w http.ResponseWriter, r *http.Request) {
	reqToken := r.Header.Get("Authorization")
	if reqToken == "" || !strings.HasPrefix(reqToken, "Bearer ") {
		sendError(w, http.StatusUnauthorized, "Please provide a valid token")
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
	issuer, err := token.Claims.GetIssuer()
	if err != nil || issuer != "chirpy-refresh" {
		sendError(w, http.StatusUnauthorized, "Please provide a refresh token")
		return
	}
	_, ok := db.Revoked[reqToken]
	if ok {
		sendError(w, http.StatusUnauthorized, "Provided token has been revoked")
		return
	}
	usrId, err := token.Claims.GetSubject()
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Getting token subject")
		fmt.Println(err)
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		Subject:   usrId,
	}).SignedString([]byte(db.jwt_secret))
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Issue creating token")
		fmt.Println(err)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"token": accessToken,
	})
}

func (db *fakeDB) revoke(w http.ResponseWriter, r *http.Request) {
	reqToken := r.Header.Get("Authorization")
	if reqToken == "" || !strings.HasPrefix(reqToken, "Bearer ") {
		sendError(w, http.StatusUnauthorized, "Please provide a valid token")
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
	issuer, err := token.Claims.GetIssuer()
	if err != nil || issuer != "chirpy-refresh" {
		sendError(w, http.StatusUnauthorized, "Please provide a refresh token")
		return
	}
	db.Revoked[reqToken] = time.Now()
}

func (db *fakeDB) redUpgrade(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("Authorization")
	if !strings.HasPrefix(apiKey, "ApiKey ") || "ApiKey "+os.Getenv("POLKA_KEY") != apiKey {
		sendError(w, http.StatusUnauthorized, "Invalid API Key")
		return
	}
	var in struct {
		Event string
		Data  struct {
			ID int `json:"user_id"`
		}
	}
	err := json.NewDecoder(r.Body).Decode(&in)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Can't parse body")
		return
	}
	if in.Event != "user.upgraded" {
		return
	}
	for e, u := range db.Users {
		if u.ID == in.Data.ID {
			db.Users[e].Red = true
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}
