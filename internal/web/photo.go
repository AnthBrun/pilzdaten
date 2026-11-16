package web

// Foto-Vorschläge: Ein Bildmodell (Repo pilz-ki) erkennt auf Fotos sichtbare
// Merkmale und füllt damit die Filter vor. Es läuft komplett im Browser
// (onnxruntime-web, WASM); Fotos verlassen das Gerät nie. Der Server liefert
// nur die Dateien aus MODEL_DIR unter /model/ aus:
//
//	MODEL_DIR/
//	├── meta.json            Eingabegröße, Tags, Schwellen, Gate, Arten (aus pilz-ki export)
//	├── model.int8.onnx      das Modell (~26 MB)
//	└── ort-<version>/       onnxruntime-web: ort.wasm.min.mjs, ort-wasm-simd-threaded.{mjs,wasm}
//
// Liegt neben einer Datei eine vorkomprimierte <datei>.gz, wird sie an
// Browser geschickt, die gzip verstehen. Fehlt MODEL_DIR oder meta.json,
// bleibt die Funktion aus und der Foto-Knopf erscheint nicht.

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// photoModel beschreibt ein gefundenes Modell. Version stammt aus meta.json
// und macht die URLs eindeutig, damit der Service Worker alte Stände ersetzt.
type photoModel struct {
	dir     string
	Version string // meta.json "version"
	Runtime string // URL-Pfad der onnxruntime-web-Dateien, z. B. /model/ort-1.31.1/
	HasGate bool   // meta.json enthält ein Gate gegen Fotos ohne Pilz
}

// loadPhotoModel liest MODEL_DIR. Kein Ordner oder kein meta.json: nil, kein Fehler.
func loadPhotoModel(dir string) (*photoModel, error) {
	if dir == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var meta struct {
		Version string          `json:"version"`
		Gate    json.RawMessage `json:"gate"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, fmt.Errorf("%s/meta.json: %w", dir, err)
	}
	if meta.Version == "" {
		return nil, fmt.Errorf("%s/meta.json: version fehlt", dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "model.int8.onnx")); err != nil {
		return nil, fmt.Errorf("%s: model.int8.onnx fehlt", dir)
	}
	runtimes, _ := filepath.Glob(filepath.Join(dir, "ort-*", "ort.wasm.min.mjs"))
	if len(runtimes) == 0 {
		return nil, fmt.Errorf("%s: onnxruntime-web fehlt (ort-<version>/ort.wasm.min.mjs)", dir)
	}
	sort.Strings(runtimes)
	runtimeDir := filepath.Base(filepath.Dir(runtimes[len(runtimes)-1]))
	hasGate := len(meta.Gate) > 0 && string(meta.Gate) != "null"
	return &photoModel{dir: dir, Version: meta.Version, Runtime: "/model/" + runtimeDir + "/", HasGate: hasGate}, nil
}

var photoMimeTypes = map[string]string{
	".mjs":  "text/javascript; charset=utf-8",
	".wasm": "application/wasm",
	".onnx": "application/octet-stream",
	".json": "application/json; charset=utf-8",
}

// serveModelFile liefert GET /model/{path...} aus MODEL_DIR, mit ETag und
// optional vorkomprimiert. "no-cache": der Browser fragt nach (304), die
// eigentliche Offline-Kopie hält der Service Worker (static/sw.js).
func (m *photoModel) serveModelFile(w http.ResponseWriter, r *http.Request) {
	rel := path.Clean("/" + r.PathValue("path"))
	if rel == "/" || strings.HasSuffix(rel, ".gz") {
		http.NotFound(w, r)
		return
	}
	file := filepath.Join(m.dir, filepath.FromSlash(rel))
	contentType := photoMimeTypes[path.Ext(rel)]
	if contentType == "" {
		contentType = mime.TypeByExtension(path.Ext(rel))
	}
	serve := file
	if acceptsGzip(r) {
		if _, err := os.Stat(file + ".gz"); err == nil {
			serve = file + ".gz"
			w.Header().Set("Content-Encoding", "gzip")
			// Ungepackte Größe für die Fortschrittsanzeige in js/photo.js.
			if original, err := os.Stat(file); err == nil {
				w.Header().Set("X-Decoded-Length", fmt.Sprint(original.Size()))
			}
		}
	}
	info, err := os.Stat(serve)
	if err != nil || info.IsDir() {
		w.Header().Del("Content-Encoding")
		http.NotFound(w, r)
		return
	}
	handle, err := os.Open(serve)
	if err != nil {
		w.Header().Del("Content-Encoding")
		http.NotFound(w, r)
		return
	}
	defer handle.Close()
	w.Header().Add("Vary", "Accept-Encoding")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", fmt.Sprintf(`"%x-%x%s"`, info.Size(), info.ModTime().UnixNano(), map[bool]string{true: "-gz"}[serve != file]))
	http.ServeContent(w, r, "", info.ModTime(), handle)
}

func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.EqualFold(strings.TrimSpace(name), "gzip") && !strings.Contains(strings.ReplaceAll(params, " ", ""), "q=0") {
			return true
		}
	}
	return false
}

// GET /api/species?slugs=a,b,c – Kartendaten für bestimmte Arten in dieser
// Reihenfolge (für die "ähnlichen Arten" der Foto-Vorschläge). Unbekannte
// Slugs fehlen in der Antwort.
func (s *Server) speciesCards(w http.ResponseWriter, r *http.Request) {
	lang := s.requestLang(w, r)
	slugs := splitTags(r.URL.Query().Get("slugs"))
	if len(slugs) > 20 {
		slugs = slugs[:20]
	}
	results := make([]searchResult, 0, len(slugs))
	for _, slug := range slugs {
		if species, ok := s.catalog.Get(slug); ok {
			results = append(results, s.cardResult(lang, species))
		}
	}
	w.Header().Add("Vary", "Cookie, Accept-Language")
	writeJSON(w, map[string][]searchResult{"results": results})
}
