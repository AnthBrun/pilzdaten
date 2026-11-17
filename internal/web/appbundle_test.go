package web

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"pilzdaten/internal/catalog"
)

func TestAppBundleNeedsToken(t *testing.T) {
	dir := t.TempDir()
	speciesPath, indexPath := filepath.Join(dir, "species.jsonl"), filepath.Join(dir, "index.jsonl")
	_ = os.WriteFile(speciesPath, []byte(`{"slug":"A","name_trivial":"A","merkmale":{},"bilddateien":[],"verwechslungen":[]}`+"\n"), 0o600)
	_ = os.WriteFile(indexPath, []byte(`{"slug":"A","name":"A"}`+"\n"), 0o600)
	data, err := catalog.Load(speciesPath, indexPath)
	if err != nil {
		t.Fatal(err)
	}
	bundleDir := filepath.Join(dir, "bundle")
	_ = os.MkdirAll(filepath.Join(bundleDir, "model-1"), 0o700)
	_ = os.WriteFile(filepath.Join(bundleDir, "manifest.json"), []byte(`{"version":"x"}`), 0o600)
	_ = os.WriteFile(filepath.Join(bundleDir, "model-1", "meta.json"), []byte(`{}`), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("geheim"), 0o600)

	off, _ := NewWithOptions(data, dir, Options{AppBundleDir: bundleDir})
	response := httptest.NewRecorder()
	off.ServeHTTP(response, httptest.NewRequest("GET", "/app/bundle/manifest.json", nil))
	if response.Code != 404 {
		t.Errorf("without tokens: status %d, want 404", response.Code)
	}

	on, _ := NewWithOptions(data, dir, Options{AppBundleDir: bundleDir, AppTokens: []string{"alt", " neu "}})
	cases := []struct {
		path, token string
		code        int
		cache       string
	}{
		{"/app/bundle/manifest.json", "", 401, ""},
		{"/app/bundle/manifest.json", "falsch", 401, ""},
		{"/app/bundle/manifest.json", "neu", 200, "private, no-cache"},
		{"/app/bundle/model-1/meta.json", "alt", 200, "private, max-age=31536000, immutable"},
		{"/app/bundle/../secret.txt", "neu", -1, ""}, // -1: alles außer 200 (der Mux leitet um)
		{"/app/bundle/fehlt.json", "neu", 404, ""},
	}
	for _, c := range cases {
		request := httptest.NewRequest("GET", c.path, nil)
		if c.token != "" {
			request.Header.Set("Authorization", "Bearer "+c.token)
		}
		response := httptest.NewRecorder()
		on.ServeHTTP(response, request)
		if (c.code == -1 && response.Code == 200) || (c.code != -1 && response.Code != c.code) {
			t.Errorf("%s (%q): status %d, want %d", c.path, c.token, response.Code, c.code)
		}
		if c.cache != "" && response.Header().Get("Cache-Control") != c.cache {
			t.Errorf("%s: Cache-Control %q", c.path, response.Header().Get("Cache-Control"))
		}
		if response.Body.String() == "geheim" {
			t.Errorf("%s leaked a file outside the bundle", c.path)
		}
	}
}
