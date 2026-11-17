// apppackage schreibt das Datenpaket der nativen App (Repo pilzkorb):
// alle Arten wie /api/offline, zusätzlich alle Fotos und großen Originale.
//
//	go run ./cmd/apppackage -data data -lang de -out app-package.json
//
// pilzkorb/tool/build_app_bundle.py nimmt die Datei und das Bildmodell
// (model.lock) und baut daraus das Bundle, das die App beim ersten Start lädt.
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"pilzdaten/internal/catalog"
	"pilzdaten/internal/i18n"
	"pilzdaten/internal/web"
)

func main() {
	dataDir := flag.String("data", "data", "Ordner mit den aufbereiteten Daten")
	lang := flag.String("lang", i18n.Source, "Sprache des Pakets")
	out := flag.String("out", "app-package.json", "Zieldatei")
	flag.Parse()

	data, err := catalog.LoadDir(*dataDir)
	if err != nil {
		log.Fatalf("Daten konnten nicht geladen werden: %v", err)
	}
	translations, err := i18n.Load(filepath.Join(*dataDir, "i18n"))
	if err != nil {
		log.Fatalf("Übersetzungen konnten nicht geladen werden: %v", err)
	}
	raw, err := web.AppPackage(data, translations, *lang)
	if err != nil {
		log.Fatalf("App-Paket konnte nicht gebaut werden: %v", err)
	}
	if err := os.WriteFile(*out, raw, 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("%d Arten, %d Bytes -> %s", data.Count(), len(raw), *out)
}
