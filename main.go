package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	debug := flag.Bool("debug", false, "debug: remove db before start")
	flag.Parse()
	err := godotenv.Load()
	if err != nil {
		log.Fatalln(err)
	}

	db, err := OpenFakeDB("database.json", *debug)
	if err != nil {
		log.Fatalln(err)
	}
	mux := http.NewServeMux()
	apiCfg := &apiConfig{}
	fileHandle := http.StripPrefix("/app", http.FileServer(http.Dir("./server")))
	mux.Handle("/app/*", apiCfg.middlewareMetricsInc(fileHandle))

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /admin/metrics/", apiCfg.metrics)
	mux.HandleFunc("/api/reset", apiCfg.reset)
	mux.HandleFunc("POST /api/chirps", db.chirp)
	mux.HandleFunc("GET /api/chirps", db.allChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", db.getChirp)
	mux.HandleFunc("POST /api/users", db.addUser)
	mux.HandleFunc("PUT /api/users", db.updateUser)
	mux.HandleFunc("POST /api/login", db.login)
	mux.HandleFunc("POST /api/refresh", db.refresh)
	mux.HandleFunc("POST /api/revoke", db.revoke)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", db.deleteChirp)

	mux.HandleFunc("POST /api/polka/webhooks", db.redUpgrade)

	serv := http.Server{
		Addr:    ":8080",
		Handler: middlewareCors(mux),
	}
	err = serv.ListenAndServe()
	fmt.Println(err)
}

func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sendError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	out, _ := json.Marshal(map[string]string{"error": msg})
	w.Write(out)
}
