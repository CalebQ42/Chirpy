package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type savedChirp struct {
	Body   string `json:"body"`
	ID     int    `json:"id"`
	Author int    `json:"author_id"`
}

func (db *fakeDB) add(chirp string, author int) (int, error) {
	db.mut.Lock()
	defer db.mut.Unlock()
	id := len(db.Chirps) + 1
	db.Chirps = append(db.Chirps, savedChirp{
		ID:     id,
		Body:   chirp,
		Author: author,
	})
	err := db.sync()
	if err != nil {
		db.Chirps = db.Chirps[:len(db.Chirps)-2]
	}
	return id, err
}

func (db *fakeDB) chirp(w http.ResponseWriter, r *http.Request) {
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
	usrID, err := token.Claims.GetSubject()
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Issue getting user ID")
		return
	}
	authID, err := strconv.Atoi(usrID)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Issue getting user ID")
		return
	}
	var in struct {
		Body string
	}
	err = json.NewDecoder(r.Body).Decode(&in)
	r.Body.Close()
	if err != nil {
		sendError(w, http.StatusBadRequest, "Something went wrong")
		return
	}
	if len(in.Body) > 140 {
		sendError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	badWords := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}
	spl := strings.Split(in.Body, " ")
	for i := range spl {
		if slices.Contains(badWords, strings.ToLower(spl[i])) {
			spl[i] = "****"
		}
	}
	in.Body = strings.Join(spl, " ")
	id, err := db.add(in.Body, authID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "error adding chirp to fakeDB")
		fmt.Println(err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	out, _ := json.Marshal(db.Chirps[id-1])
	w.Write(out)
}

func (db *fakeDB) getChirp(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("chirpID"))
	if err != nil || id > len(db.Chirps) {
		sendError(w, http.StatusNotFound, "Invalid Chirp ID")
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(db.Chirps[id-1])
}

func (db *fakeDB) allChirps(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(db.Chirps)
}

func (db *fakeDB) deleteChirp(w http.ResponseWriter, r *http.Request) {
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
	usrID, err := token.Claims.GetSubject()
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Issue getting user ID")
		return
	}
	authID, err := strconv.Atoi(usrID)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Issue getting user ID")
		return
	}
	chirpIDStr := r.PathValue("chirpID")
	if chirpIDStr == "" {
		sendError(w, http.StatusBadRequest, "Please provide a chirp ID")
		return
	}
	chirpID, err := strconv.Atoi(chirpIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Chirp ID is invalid")
		return
	}
	found := false
	for i := range db.Chirps {
		if db.Chirps[i].ID == chirpID {
			found = true
			if db.Chirps[i].Author != authID {
				sendError(w, http.StatusForbidden, "You don't own this chirp")
				return
			}
			db.Chirps = append(db.Chirps[:i], db.Chirps[i+1:]...)
			break
		}
	}
	if !found {
		sendError(w, http.StatusBadRequest, "Chirp ID is invalid")
		return
	}
}
