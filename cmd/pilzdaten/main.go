package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"pilzdaten/internal/catalog"
	"pilzdaten/internal/web"
)

// Konfiguration über Umgebungsvariablen:
//
//	DATA_DIR   Ordner mit den aufbereiteten Daten (Standard: ./data im Repo)
//	IMAGE_DIR  Wurzel der Artfotos, z. B. …/pilzsuche/raw/images (Pflicht)
//	ADDR       Listen-Adresse (Standard: :8080)
func main() {
	dataDir := envOr("DATA_DIR", "data")
	imageDir := os.Getenv("IMAGE_DIR")
	if imageDir == "" {
		log.Fatal("IMAGE_DIR ist nicht gesetzt: Pfad zum Bilderordner angeben (siehe README, \"Bilder\")")
	}
	if info, err := os.Stat(imageDir); err != nil || !info.IsDir() {
		log.Fatalf("IMAGE_DIR %q ist kein lesbarer Ordner", imageDir)
	}

	file := func(name string) string { return filepath.Join(dataDir, name) }
	data, err := catalog.LoadWithAssets(
		file("species.jsonl"), file("index_arts.jsonl"), file("filter_tags.jsonl"),
		file("facets.json"), file("best_images.jsonl"), file("pictogram_map.json"),
	)
	if err != nil {
		log.Fatalf("Daten konnten nicht geladen werden: %v", err)
	}
	log.Printf("%d Arten aus %s geladen, Bilder aus %s", data.Count(), dataDir, imageDir)

	mux := web.New(data, imageDir)
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
