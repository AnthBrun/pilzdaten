package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"pilzdaten/internal/catalog"
	"pilzdaten/internal/i18n"
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

	data, err := catalog.LoadDir(dataDir)
	if err != nil {
		log.Fatalf("Daten konnten nicht geladen werden: %v", err)
	}
	translations, err := i18n.Load(filepath.Join(dataDir, "i18n"))
	if err != nil {
		log.Fatalf("Übersetzungen konnten nicht geladen werden: %v", err)
	}
	languages := make([]string, 0)
	for _, lang := range translations.Languages() {
		languages = append(languages, lang.Code)
	}
	log.Printf("%d Arten aus %s geladen, Bilder aus %s, Sprachen: %s", data.Count(), dataDir, imageDir, strings.Join(languages, ", "))

	mux := web.NewWithTranslations(data, imageDir, translations)
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
