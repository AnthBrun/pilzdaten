package main

import (
	"log"
	"net/http"
	"os"

	"pilzdaten/internal/catalog"
	"pilzdaten/internal/web"
)

func main() {
	speciesPath := envOr("SPECIES_FILE", "/data/species.jsonl")
	indexPath := envOr("INDEX_FILE", "/data/index_arts.jsonl")
	filterTagsPath := envOr("FILTER_TAGS_FILE", "/data/filter_tags.jsonl")
	facetsPath := envOr("FACETS_FILE", "/data/facets.json")
	data, err := catalog.LoadWithFilters(speciesPath, indexPath, filterTagsPath, facetsPath)
	if err != nil {
		log.Fatalf("Daten konnten nicht geladen werden: %v", err)
	}
	log.Printf("%d Arten aus %s geladen", data.Count(), speciesPath)

	mux := web.New(data, envOr("IMAGE_DIR", "/images"))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
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
