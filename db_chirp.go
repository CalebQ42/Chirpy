package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type savedChirp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

type fakeDB struct {
	mut    *sync.RWMutex
	Chirps []savedChirp
	Users  map[string]user
	path   string
}

func OpenFakeDB(file string) *fakeDB {
	os.Remove(file) //theoretically replace this with actually loading the values, but for this assignment (at this point) it doesn't matter.
	return &fakeDB{
		mut:   &sync.RWMutex{},
		path:  file,
		Users: make(map[string]user),
	}
}

func (db *fakeDB) sync() error {
	os.Remove(db.path)
	fil, err := os.Create(db.path)
	if err != nil {
		return err
	}
	return json.NewEncoder(fil).Encode(db)
}

func (db *fakeDB) add(chirp string) (int, error) {
	db.mut.Lock()
	defer db.mut.Unlock()
	id := len(db.Chirps) + 1
	db.Chirps = append(db.Chirps, savedChirp{
		ID:   id,
		Body: chirp,
	})
	err := db.sync()
	if err != nil {
		db.Chirps = db.Chirps[:len(db.Chirps)-2]
	}
	return id, err
}

func (db *fakeDB) chirp(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Body string
	}
	err := json.NewDecoder(r.Body).Decode(&in)
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

	id, err := db.add(in.Body)
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
