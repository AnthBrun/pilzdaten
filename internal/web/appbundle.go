package web

import (
	"crypto/subtle"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// appBundle liefert das Daten-Bundle der nativen App (internal/appbundle)
// unter /app/bundle/ aus. Nur mit gültigem Token (Authorization: Bearer …);
// ohne Tokens oder ohne Bundle ist newAppBundle nil und die Route liefert 404.
type appBundle struct {
	dir    string
	tokens [][]byte
}

func newAppBundle(dir string, tokens []string) *appBundle {
	if dir == "" {
		return nil
	}
	if _, err := os.Stat(filepath.Join(dir, "manifest.json")); err != nil {
		return nil
	}
	bundle := &appBundle{dir: dir}
	for _, token := range tokens {
		if token = strings.TrimSpace(token); token != "" {
			bundle.tokens = append(bundle.tokens, []byte(token))
		}
	}
	if len(bundle.tokens) == 0 {
		return nil
	}
	return bundle
}

func (b *appBundle) authorized(r *http.Request) bool {
	given, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return false
	}
	valid := false
	for _, token := range b.tokens { // alle vergleichen, gleich lange Laufzeit
		if subtle.ConstantTimeCompare([]byte(given), token) == 1 {
			valid = true
		}
	}
	return valid
}

func (b *appBundle) serve(w http.ResponseWriter, r *http.Request) {
	if b == nil {
		http.NotFound(w, r)
		return
	}
	if !b.authorized(r) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="pilzkorb"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	rel := path.Clean("/" + r.PathValue("path"))
	if rel == "/" {
		http.NotFound(w, r)
		return
	}
	file := filepath.Join(b.dir, filepath.FromSlash(rel))
	info, err := os.Stat(file)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	// manifest.json ändert sich, alles andere trägt die Version im Namen.
	if rel == "/manifest.json" {
		w.Header().Set("Cache-Control", "private, no-cache")
	} else {
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	}
	w.Header().Set("Vary", "Authorization")
	http.ServeFile(w, r, file)
}
