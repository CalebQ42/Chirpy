package main

import (
	"encoding/json"
	"os"
	"sync"
)

type fakeDB struct {
	mut        *sync.RWMutex
	Users      map[string]user
	path       string
	jwt_secret string
	Chirps     []savedChirp
}

func OpenFakeDB(file string, debug bool) (*fakeDB, error) {
	if debug {
		os.Remove(file) //theoretically replace this with actually loading the values, but for this assignment (at this point) it doesn't matter.
	}
	fil, err := os.Open(file)
	if os.IsNotExist(err) {
		fil, err = os.Create(file)
		if err != nil {
			return nil, err
		}
		return &fakeDB{
			mut:        &sync.RWMutex{},
			path:       file,
			Users:      make(map[string]user),
			jwt_secret: os.Getenv("JWT_SECRET"),
		}, nil
	}
	return load(fil)
}

func load(fil *os.File) (*fakeDB, error) {
	var mp map[string]any
	err := json.NewDecoder(fil).Decode(&mp)
	if err != nil {
		return nil, err
	}
	db := &fakeDB{
		mut:        &sync.RWMutex{},
		path:       fil.Name(),
		Users:      make(map[string]user),
		Chirps:     mp["chirps"].([]savedChirp),
		jwt_secret: os.Getenv("JWT_SECRET"),
	}
	for _, u := range mp["users"].([]user) {
		db.Users[u.Email] = u
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
		"users":  usrOut,
		"chirps": db.Chirps,
	}
	return json.Marshal(mp)
}
