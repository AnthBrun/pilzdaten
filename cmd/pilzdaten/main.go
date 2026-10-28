package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"pilzdaten/internal/catalog"
)

func main() {
	speciesPath := envOr("SPECIES_FILE", "/data/species.jsonl")
	indexPath := envOr("INDEX_FILE", "/data/index_arts.jsonl")
	data, err := catalog.Load(speciesPath, indexPath)
	if err != nil {
		log.Fatalf("Daten konnten nicht geladen werden: %v", err)
	}
	log.Printf("%d Arten aus %s geladen", data.Count(), speciesPath)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, "<!doctype html><html lang=\"de\"><meta charset=\"utf-8\"><title>Pilzdatenbank</title><h1>Pilzdatenbank</h1>")
	})

	addr := envOr("ADDR", ":8080")
	log.Printf("HTTP-Server hört auf %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
