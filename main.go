package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type apiConfig struct {
	fileserverHits int
	mu             sync.Mutex
}

type ChirpRequest struct {
	Body string 'json:"body"'
}

type ErrorResponse struct {
	Error string 'json:"error"'
}

type ValidResponse struct {
	Valid bool 'json:"valid"'
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
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Hits: %d\n", cfg.fileserverHits)
}

func (cfg *apiConfig) adminMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	w.Header().Set("Content-Type", "Text/html; charset=utf-8")
	fmt.Fprintf(w, `
	<html>
	<body>
		<h1>Welcome, Chirpy Admin<h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
	</html>
	`, cfg.fileserverHits)
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.fileserverHits = 0
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0\n"))
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var chirpRequest ChirpRequest
	err := json.NewDecoder(r.Body).Decode(&chirpRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encdoe(ErrorResponse{Error: "Invalid request body"})
		return
	}

	if len(chirpRequest.Body) > 140 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Chirp is too long"})
		return
	}

	if len(chirpRequest.Body) > 140 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Chirp is too long"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ValidResponse{Valid: true})
}

func main() {
	const port = "8080"

	apiCfg := &apiConfig{}

	mux := http.NewServeMux()

	// readiness endpoint handler
	mux.HandleFunc("/api/healthz", healthzHandler)

	fileServer := http.FileServer(http.Dir("."))
	// File server for the /app/* path
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fileServer)))

	// admin metrics endpoint handler
	mux.HandleFunc("/admin/metrics", apiCfg.adminMetricsHandler)

	// metrics endpoint handler
	mux.HandleFunc("/api/metrics", apiCfg.metricsHandler)

	// reset endpoint handler
	mux.HandleFunc("/api/reset", apiCfg.resetHandler)

	mux.Handle("/", fileServer)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}
