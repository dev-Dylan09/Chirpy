package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

type apiConfig struct {
	fileserverHits int
	mu             sync.Mutex
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.mu.Lock()
		cfg.fileserverHits++
		cfg.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Hits: %d\n", cfg.fileserverHits)
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.fileserverHits = 0
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0\n"))
}

func main() {
	const port = "8080"

	apiCfg := &apiConfig{}

	mux := http.NewServeMux()

	// Readiness endpoint handler
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fileServer := http.FileServer(http.Dir("."))
	// File server for the /app/* path
	mux.Handle("/app/", apiCFG.middlewareMetricsInc(http.StripPrefix("/app", fileServer)))

	// metrics endpoint handler
	mux.HandleFunc("/metrics", apiCfg.metricsHandler)

	// reset endpoint handler
	mux.HandleFunc("/reset", apiCfg.resetHandler)

	mux.Handle("/", fileServer)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}
