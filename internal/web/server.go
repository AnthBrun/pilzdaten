// Package web liefert die Pilzdatenbank aus: HTML-Seiten (home.go, detail.go),
// JSON-API (api.go), Offline-Paket und Service Worker (offline.go) sowie die
// eingebetteten Templates und statischen Dateien.
package web

import (
	"embed"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"pilzdaten/internal/catalog"
	"pilzdaten/internal/i18n"
)

//go:embed templates/*.html static/*
var files embed.FS

// pageSize ist die Trefferzahl pro Seite für Startseite und Such-API.
const pageSize = 50

type Server struct {
	catalog   *catalog.Catalog
	i18n      *i18n.Bundle
	templates map[string]*template.Template // je Sprache, mit sprachgebundenem t/tf
	offlineMu sync.Mutex
	offline   map[string]*offlineData // Offline-Paket je Sprache
}

// New baut den HTTP-Router. imageDir ist das Wurzelverzeichnis der Artfotos,
// es wird unter /media/ (Vorschauen) und /original/ (große Originale) ausgeliefert
// (siehe README, "Bilder").
func New(data *catalog.Catalog, imageDir string) *http.ServeMux {
	return NewWithTranslations(data, imageDir, nil)
}

// NewWithTranslations wie New, mit Übersetzungen (nil = nur Deutsch).
// Die übersetzten Trivialnamen werden zusätzlich durchsuchbar gemacht.
func NewWithTranslations(data *catalog.Catalog, imageDir string, bundle *i18n.Bundle) *http.ServeMux {
	server := &Server{
		catalog:   data,
		i18n:      bundle,
		templates: map[string]*template.Template{},
		offline:   map[string]*offlineData{},
	}
	for _, lang := range bundle.Languages() {
		server.templates[lang.Code] = template.Must(template.New("pages").Funcs(templateFuncs).Funcs(server.langFuncs(lang.Code)).ParseFS(files, "templates/*.html"))
		if lang.Code != i18n.Source {
			code := lang.Code
			data.SetLocalNames(code, func(text string) string { return bundle.T(code, text) })
		}
	}
	noStore := func(string) string { return "no-store" }
	oneDay := func(string) string { return "public, max-age=86400" }

	mux := http.NewServeMux()
	// Seiten
	mux.HandleFunc("GET /", server.home)
	mux.HandleFunc("GET /art/{slug}", server.detail)
	mux.HandleFunc("GET /offline", server.offlinePage)
	// API
	mux.Handle("GET /api/search", withCacheControl(http.HandlerFunc(server.search), noStore))
	mux.Handle("GET /api/available-tags", withCacheControl(http.HandlerFunc(server.availableTags), noStore))
	mux.HandleFunc("GET /api/facets", server.facets)
	mux.HandleFunc("GET /api/offline", server.offlinePackage)
	// PWA
	mux.HandleFunc("GET /sw.js", serviceWorker)
	mux.HandleFunc("GET /manifest.webmanifest", manifest)
	// Dateien
	mux.Handle("GET /static/", withCacheControl(http.FileServer(http.FS(files)), func(path string) string {
		if strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".js") {
			return "no-cache" // Revalidieren (ETag), damit Updates auf dem Handy ankommen
		}
		return "public, max-age=86400"
	}))
	mux.Handle("GET /media/", http.StripPrefix("/media/", withCacheControl(http.FileServer(http.Dir(imageDir)), oneDay)))
	mux.Handle("GET /original/", http.StripPrefix("/original/", withCacheControl(http.FileServer(http.Dir(imageDir)), oneDay)))
	return mux
}

func (s *Server) render(w http.ResponseWriter, lang, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Language", lang)
	w.Header().Add("Vary", "Cookie, Accept-Language")
	if err := s.templates[lang].ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, s.t(lang, msgRenderFailed), http.StatusInternalServerError)
	}
}

var msgRenderFailed = i18n.N("Seite konnte nicht gerendert werden")

// --- Sprache

const langCookie = "lang"

// requestLang wählt die Sprache: ?lang= (wird als Cookie gemerkt), dann das
// Cookie, dann Accept-Language, sonst Deutsch.
func (s *Server) requestLang(w http.ResponseWriter, r *http.Request) string {
	if code := r.URL.Query().Get("lang"); code != "" && s.i18n.Has(code) {
		http.SetCookie(w, &http.Cookie{Name: langCookie, Value: code, Path: "/", MaxAge: 365 * 24 * 3600, SameSite: http.SameSiteLaxMode})
		return code
	}
	if cookie, err := r.Cookie(langCookie); err == nil && s.i18n.Has(cookie.Value) {
		return cookie.Value
	}
	return s.i18n.Match(r.Header.Get("Accept-Language"))
}

// t übersetzt einen deutschen Text in die Sprache lang.
func (s *Server) t(lang, text string) string {
	return s.i18n.T(lang, text)
}

func withCacheControl(next http.Handler, policy func(path string) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", policy(r.URL.Path))
		next.ServeHTTP(w, r)
	})
}

// --- Gemeinsame Query-Parameter von Startseite und API:
// q (Name), f (Tags, kommagetrennt), dmin/dmax (Hutdurchmesser in cm),
// page, letter (A–Z oder #).

func queryInt(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	return value
}

func queryPage(r *http.Request) int {
	return max(1, queryInt(r, "page", 1))
}

// diameterRange liest dmin/dmax; Werte außerhalb des Datenbereichs fallen
// auf die Grenzen zurück.
func diameterRange(r *http.Request, facets catalog.Facets) (int, int) {
	bounded := func(key string, fallback int) int {
		value := queryInt(r, key, fallback)
		if value < facets.MinDiameterCM || value > facets.MaxDiameterCM {
			return fallback
		}
		return value
	}
	minCM, maxCM := bounded("dmin", facets.MinDiameterCM), bounded("dmax", facets.MaxDiameterCM)
	if minCM > maxCM {
		minCM, maxCM = maxCM, minCM
	}
	return minCM, maxCM
}

func validLetter(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "#" || len(value) == 1 && value[0] >= 'A' && value[0] <= 'Z' {
		return value
	}
	return ""
}

func splitTags(value string) []string {
	seen := make(map[string]struct{})
	tags := make([]string, 0)
	for _, tag := range strings.Split(value, ",") {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; !ok {
			seen[tag] = struct{}{}
			tags = append(tags, tag)
		}
	}
	return tags
}
