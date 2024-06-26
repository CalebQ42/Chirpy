package main

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

type fakeDB struct {
	mut        *sync.RWMutex
	Users      map[string]*user
	Revoked    map[string]time.Time
	path       string
	jwt_secret string
	Chirps     []savedChirp
}

func OpenFakeDB(file string, debug bool) (*fakeDB, error) {
	if os.Getenv("JWT_SECRET") == "" {
		return nil, errors.New("JWT_SECRET is not defined")
	}
	if debug {
		os.Remove(file) //theoretically replace this with actually loading the values, but for this assignment (at this point) it doesn't matter.
	}
	fil, err := os.Open(file)
	if os.IsNotExist(err) {
		return &fakeDB{
			mut:        &sync.RWMutex{},
			path:       file,
			Users:      make(map[string]*user),
			jwt_secret: os.Getenv("JWT_SECRET"),
			Revoked:    make(map[string]time.Time),
		}, nil
	}
	return load(fil)
}

func load(fil *os.File) (*fakeDB, error) {
	// This is actually terrible, don't look at it or you might die or something.
	var mp map[string]any
	err := json.NewDecoder(fil).Decode(&mp)
	if err != nil {
		return nil, err
	}
	var chirps []savedChirp
	if mp["chirps"] != nil {
		chirps = make([]savedChirp, len(mp["chirps"].([]any)))
		for i, c := range mp["chirps"].([]any) {
			cMp := c.(map[string]any)
			chirps[i] = savedChirp{
				ID:   int(cMp["id"].(float64)),
				Body: cMp["body"].(string),
			}
		}
	}
	db := &fakeDB{
		mut:        &sync.RWMutex{},
		path:       fil.Name(),
		Users:      make(map[string]*user),
		Chirps:     chirps,
		jwt_secret: os.Getenv("JWT_SECRET"),
		Revoked:    make(map[string]time.Time),
	}
	if mp["users"] != nil {
		for _, u := range mp["users"].([]any) {
			uMp := u.(map[string]any)
			db.Users[uMp["email"].(string)] = &user{
				Email:    uMp["email"].(string),
				password: uMp["password"].(string),
				ID:       int(uMp["id"].(float64)),
			}
		}
	}
	if mp["revoked"] != nil {
		rev := mp["revoked"].(map[string]any)
		for k := range rev {
			db.Revoked[k] = rev[k].(time.Time)
		}
	}
	return db, nil
}

func (db *fakeDB) sync() error {
	os.Remove(db.path)
	fil, err := os.Create(db.path)
	if err != nil {
		return err
	}
	return json.NewEncoder(fil).Encode(db)
}

func (db *fakeDB) MarshalJSON() ([]byte, error) {
	usrOut := make([]map[string]any, len(db.Users))
	indx := 0
	for _, u := range db.Users {
		usrOut[indx] = u.fullMarshal()
		indx++
	}
	mp := map[string]any{
		"users":   usrOut,
		"chirps":  db.Chirps,
		"revoked": db.Revoked,
	}
	return json.Marshal(mp)
}
