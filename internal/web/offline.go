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
	"pilzdaten/internal/i18n"
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
	Lang    string           `json:"lang"`
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

// buildOfflinePackage baut das Paket einer Sprache beim ersten Abruf.
func (s *Server) buildOfflinePackage(lang string) ([]byte, string, error) {
	s.offlineMu.Lock()
	cached := s.offline[lang]
	if cached == nil {
		cached = &offlineData{}
		s.offline[lang] = cached
	}
	s.offlineMu.Unlock()
	cached.once.Do(func() {
		pkg := offlinePackage{Lang: lang, Facets: s.localFacets(lang)}
		for _, species := range s.catalog.All() {
			pkg.Species = append(pkg.Species, s.offlineSpeciesFor(lang, species))
		}
		raw, err := json.Marshal(pkg)
		if err != nil {
			cached.err = err
			return
		}
		sum := sha256.Sum256(raw)
		version := hex.EncodeToString(sum[:8])
		raw = bytes.Replace(raw, []byte(`"version":""`), []byte(`"version":"`+version+`"`), 1)
		var buffer bytes.Buffer
		writer, _ := gzip.NewWriterLevel(&buffer, gzip.BestCompression)
		if _, err := writer.Write(raw); err != nil {
			cached.err = err
			return
		}
		if err := writer.Close(); err != nil {
			cached.err = err
			return
		}
		cached.gzipped = buffer.Bytes()
		cached.etag = `"` + version + `"`
	})
	return cached.gzipped, cached.etag, cached.err
}

func (s *Server) offlineSpeciesFor(lang string, species *catalog.Species) offlineSpecies {
	filterData := s.catalog.FilterData(species.Slug)
	localNames := s.t(lang, species.NameTrivial)
	names := localNames
	if localNames != species.NameTrivial {
		names += ", " + species.NameTrivial // offline auch nach deutschen Namen suchen
	}
	entry := offlineSpecies{
		Slug: species.Slug, Name: displayName(localNames), Names: names,
		Scientific: species.NameWissenschaftlich, Synonyms: species.Synonyme, Genus: s.t(lang, species.Gattung),
		Edibility: species.EdibilityClass, EdibilityTxt: s.t(lang, species.EdibilityLabel),
		Tags: filterData.Tags, Diameter: [2]int{filterData.MinCM, filterData.MaxCM},
		Priority: species.Priority, Commonness: species.Commonness, Photos: species.PhotoCount,
		Related: species.Verwechslungen, Source: species.URL,
	}
	if entry.Tags == nil {
		entry.Tags = []string{}
	}
	for _, image := range speciesPhotos(species, noOriginal) {
		if entry.Image == "" {
			entry.Image = image.URL
		}
		if len(entry.Images) < 8 {
			entry.Images = append(entry.Images, image.URL)
		}
	}
	main, supplemental := speciesTraits(species)
	main, supplemental = s.localTraits(lang, main), s.localTraits(lang, supplemental)
	for _, value := range main {
		entry.Traits = append(entry.Traits, [2]string{value.Name, value.Value})
	}
	for _, value := range supplemental {
		entry.Extra = append(entry.Extra, [2]string{value.Name, value.Value})
	}
	return entry
}

var msgPackFailed = i18n.N("Offline-Paket konnte nicht erstellt werden")

func (s *Server) offlinePackage(w http.ResponseWriter, r *http.Request) {
	lang := s.requestLang(w, r)
	gzipped, etag, err := s.buildOfflinePackage(lang)
	if err != nil {
		http.Error(w, s.t(lang, msgPackFailed), http.StatusInternalServerError)
		return
	}
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Vary", "Accept-Encoding, Cookie, Accept-Language")
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
		http.Error(w, s.t(lang, msgPackFailed), http.StatusInternalServerError)
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

// offlinePage ist die Hülle, die der Service Worker für nicht gespeicherte
// Artseiten ausliefert; js/offline.js rendert die Art aus dem Offline-Paket.
func (s *Server) offlinePage(w http.ResponseWriter, r *http.Request) {
	s.render(w, s.requestLang(w, r), "offline.html", nil)
}

// noOriginal: Das Offline-Paket enthält nur Vorschaubilder, keine großen Originale.
func noOriginal(string) string { return "" }
