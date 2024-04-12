package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	apiCfg := &apiConfig{}
	fileHandle := http.StripPrefix("/app", http.FileServer(http.Dir("./server")))
	mux.Handle("/app/*", apiCfg.middlewareMetricsInc(fileHandle))
	mux.Handle("GET /metrics/", apiCfg.metrics())
	mux.Handle("/reset", apiCfg.reset())
	serv := http.Server{
		Addr:    ":8080",
		Handler: middlewareCors(mux),
	}
	err := serv.ListenAndServe()
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
