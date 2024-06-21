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
