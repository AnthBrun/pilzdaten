// apppackage baut das Daten-Bundle der nativen App (Repo pilzkorb).
//
//	go run ./cmd/apppackage -data data -model data/model -bundle app-bundle
//	go run ./cmd/apppackage -data data -out app-package.json   (nur die Artdaten)
//
// Das Docker-Image baut das Bundle selbst (Stage "appbundle") und liefert es
// unter /app/bundle/ aus, sobald APP_TOKENS gesetzt ist (siehe README).
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"pilzdaten/internal/appbundle"
	"pilzdaten/internal/catalog"
	"pilzdaten/internal/i18n"
	"pilzdaten/internal/web"
)

func main() {
	dataDir := flag.String("data", "data", "Ordner mit den aufbereiteten Daten")
	lang := flag.String("lang", i18n.Source, "Sprache des Pakets")
	out := flag.String("out", "", "nur das App-Paket (JSON) in diese Datei schreiben")
	bundle := flag.String("bundle", "", "komplettes Bundle in diesen Ordner schreiben")
	modelDir := flag.String("model", "data/model", "Bildmodell (meta.json, model.int8.onnx) für -bundle")
	flag.Parse()
	if (*out == "") == (*bundle == "") {
		log.Fatal("genau eins von -out oder -bundle angeben")
	}

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
	if *out != "" {
		if err := os.WriteFile(*out, raw, 0o644); err != nil {
			log.Fatal(err)
		}
		log.Printf("%d Arten, %d Bytes -> %s", data.Count(), len(raw), *out)
		return
	}
	manifest, err := appbundle.Write(*bundle, raw, *modelDir)
	if err != nil {
		log.Fatalf("Bundle konnte nicht gebaut werden: %v", err)
	}
	log.Printf("Bundle %s (%d Arten) -> %s", manifest.Version, data.Count(), *bundle)
}
