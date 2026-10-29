package web

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"pilzdaten/internal/catalog"
)

// offlineSpecies ist die kompakte Form einer Art im Offline-Paket. Kurze
// JSON-Schlüssel halten das Paket für den Download im Wald klein.
type offlineSpecies struct {
	Slug         string      `json:"s"`
	Name         string      `json:"n"`
	Names        string      `json:"a"`
	Scientific   string      `json:"w"`
	Synonyms     []string    `json:"y,omitempty"`
	Genus        string      `json:"g,omitempty"`
	Edibility    string      `json:"e"`
	EdibilityTxt string      `json:"l"`
	Image        string      `json:"i,omitempty"`
	Images       []string    `json:"x,omitempty"`
	Tags         []string    `json:"t"`
	Diameter     [2]int      `json:"d"`
	Priority     int         `json:"p"`
	Commonness   int         `json:"c,omitempty"`
	Photos       int         `json:"k,omitempty"`
	Traits       [][2]string `json:"m,omitempty"`
	Extra        [][2]string `json:"z,omitempty"`
	Related      []string    `json:"r,omitempty"`
	Source       string      `json:"u,omitempty"`
}

type offlinePackage struct {
	Version string           `json:"version"`
	Facets  catalog.Facets   `json:"facets"`
	Species []offlineSpecies `json:"species"`
}

// offlineData baut das Paket einmal beim ersten Abruf und hält es gzip-
// komprimiert im Speicher; der Katalog ändert sich zur Laufzeit nicht.
type offlineData struct {
	once    sync.Once
	gzipped []byte
	etag    string
	err     error
}

func (s *Server) buildOfflinePackage() ([]byte, string, error) {
	s.offline.once.Do(func() {
		pkg := offlinePackage{Facets: s.catalog.Facets()}
		for _, species := range s.catalog.All() {
			pkg.Species = append(pkg.Species, offlineSpeciesFor(s.catalog, species))
		}
		raw, err := json.Marshal(pkg)
		if err != nil {
			s.offline.err = err
			return
		}
		sum := sha256.Sum256(raw)
		version := hex.EncodeToString(sum[:8])
		raw = bytes.Replace(raw, []byte(`"version":""`), []byte(`"version":"`+version+`"`), 1)
		var buffer bytes.Buffer
		writer, _ := gzip.NewWriterLevel(&buffer, gzip.BestCompression)
		if _, err := writer.Write(raw); err != nil {
			s.offline.err = err
			return
		}
		if err := writer.Close(); err != nil {
			s.offline.err = err
			return
		}
		s.offline.gzipped = buffer.Bytes()
		s.offline.etag = `"` + version + `"`
	})
	return s.offline.gzipped, s.offline.etag, s.offline.err
}

func offlineSpeciesFor(data *catalog.Catalog, species *catalog.Species) offlineSpecies {
	filterData := data.FilterData(species.Slug)
	entry := offlineSpecies{
		Slug: species.Slug, Name: displayName(species.NameTrivial), Names: species.NameTrivial,
		Scientific: species.NameWissenschaftlich, Synonyms: species.Synonyme, Genus: species.Gattung,
		Edibility: species.EdibilityClass, EdibilityTxt: species.EdibilityLabel,
		Tags: filterData.Tags, Diameter: [2]int{filterData.MinCM, filterData.MaxCM},
		Priority: species.Priority, Commonness: species.Commonness, Photos: species.PhotoCount,
		Related: species.Verwechslungen, Source: species.URL,
	}
	if entry.Tags == nil {
		entry.Tags = []string{}
	}
	for _, image := range speciesPhotos(species) {
		if entry.Image == "" {
			entry.Image = image.URL
		}
		if len(entry.Images) < 8 {
			entry.Images = append(entry.Images, image.URL)
		}
	}
	main, supplemental := speciesTraits(species)
	for _, value := range main {
		entry.Traits = append(entry.Traits, [2]string{value.Name, value.Value})
	}
	for _, value := range supplemental {
		entry.Extra = append(entry.Extra, [2]string{value.Name, value.Value})
	}
	return entry
}

func (s *Server) offlinePackage(w http.ResponseWriter, r *http.Request) {
	gzipped, etag, err := s.buildOfflinePackage()
	if err != nil {
		http.Error(w, "Offline-Paket konnte nicht erstellt werden", http.StatusInternalServerError)
		return
	}
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Vary", "Accept-Encoding")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(gzipped)
		return
	}
	reader, err := gzip.NewReader(bytes.NewReader(gzipped))
	if err != nil {
		http.Error(w, "Offline-Paket konnte nicht gelesen werden", http.StatusInternalServerError)
		return
	}
	defer reader.Close()
	_, _ = io.Copy(w, reader)
}

// serviceWorker liefert sw.js vom Wurzelpfad aus, damit der Service Worker
// die ganze Seite (Start, Artseiten, Bilder) offline bedienen darf.
func serviceWorker(w http.ResponseWriter, _ *http.Request) {
	script, err := files.ReadFile("static/sw.js")
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(script)
}

func manifest(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/manifest+json")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name":             "Pilzdatenbank – Pilze im Wald nachschlagen",
		"short_name":       "Pilze",
		"lang":             "de",
		"start_url":        "/",
		"scope":            "/",
		"display":          "standalone",
		"background_color": "#f5f4ef",
		"theme_color":      "#254e3b",
		"icons": []map[string]string{
			{"src": "/static/icon-192.png", "sizes": "192x192", "type": "image/png"},
			{"src": "/static/icon-512.png", "sizes": "512x512", "type": "image/png"},
			{"src": "/static/icon-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable"},
		},
	})
}
