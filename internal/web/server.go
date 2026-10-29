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

	"pilzdaten/internal/catalog"
)

//go:embed templates/*.html static/*
var files embed.FS

// pageSize ist die Trefferzahl pro Seite für Startseite und Such-API.
const pageSize = 50

type Server struct {
	catalog   *catalog.Catalog
	templates *template.Template
	offline   offlineData
}

// New baut den HTTP-Router. imageDir ist das Wurzelverzeichnis der Artfotos,
// es wird unter /media/ ausgeliefert (siehe README, "Bilder").
func New(data *catalog.Catalog, imageDir string) *http.ServeMux {
	server := &Server{
		catalog:   data,
		templates: template.Must(template.New("pages").Funcs(templateFuncs).ParseFS(files, "templates/*.html")),
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
	return mux
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Seite konnte nicht gerendert werden", http.StatusInternalServerError)
	}
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
