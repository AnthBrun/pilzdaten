package web

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pilzdaten/internal/catalog"
)

func photoTestCatalog(t *testing.T) (*catalog.Catalog, string) {
	t.Helper()
	dir := t.TempDir()
	species := `{"slug":"Steinpilz","name_trivial":"Steinpilz","name_wissenschaftlich":"BOLETUS EDULIS","merkmale":{},"bilddateien":["daten/s.jpg"],"verwechslungen":[]}
{"slug":"Fliegenpilz","name_trivial":"Fliegenpilz","name_wissenschaftlich":"AMANITA MUSCARIA","merkmale":{},"bilddateien":[],"verwechslungen":[]}
`
	index := `{"slug":"Steinpilz","name":"Steinpilz","wissenschaftlich":"BOLETUS EDULIS","gattung":"Röhrlinge"}
{"slug":"Fliegenpilz","name":"Fliegenpilz","wissenschaftlich":"AMANITA MUSCARIA","gattung":"Wulstlinge"}
`
	writeTestFile(t, filepath.Join(dir, "species.jsonl"), species)
	writeTestFile(t, filepath.Join(dir, "index.jsonl"), index)
	loaded, err := catalog.Load(filepath.Join(dir, "species.jsonl"), filepath.Join(dir, "index.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return loaded, dir
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeModelDir(t *testing.T, dir string) {
	t.Helper()
	writeTestFile(t, filepath.Join(dir, "meta.json"), `{"version":"20261116-1","gate":{"output":"fungus_prob","threshold":0.23}}`)
	writeTestFile(t, filepath.Join(dir, "model.int8.onnx"), "onnx-bytes")
	writeTestFile(t, filepath.Join(dir, "ort-1.31.1", "ort.wasm.min.mjs"), "export {};")
	writeTestFile(t, filepath.Join(dir, "ort-1.31.1", "ort-wasm-simd-threaded.wasm"), strings.Repeat("w", 4000))
	var packed bytes.Buffer
	zw := gzip.NewWriter(&packed)
	_, _ = zw.Write([]byte(strings.Repeat("w", 4000)))
	_ = zw.Close()
	writeTestFile(t, filepath.Join(dir, "ort-1.31.1", "ort-wasm-simd-threaded.wasm.gz"), packed.String())
}

func TestLoadPhotoModel(t *testing.T) {
	if model, err := loadPhotoModel(""); model != nil || err != nil {
		t.Fatalf("empty dir: %v %v", model, err)
	}
	if model, err := loadPhotoModel(t.TempDir()); model != nil || err != nil {
		t.Fatalf("dir without meta.json must disable the feature: %v %v", model, err)
	}
	incomplete := t.TempDir()
	writeTestFile(t, filepath.Join(incomplete, "meta.json"), `{"version":"1"}`)
	if _, err := loadPhotoModel(incomplete); err == nil {
		t.Fatal("meta.json without model must be an error")
	}
	dir := t.TempDir()
	writeModelDir(t, dir)
	model, err := loadPhotoModel(dir)
	if err != nil {
		t.Fatal(err)
	}
	if model.Version != "20261116-1" || model.Runtime != "/model/ort-1.31.1/" || !model.HasGate {
		t.Fatalf("unexpected model %+v", model)
	}
}

func TestPhotoButtonOnlyWithModel(t *testing.T) {
	loaded, imageDir := photoTestCatalog(t)
	without := httptest.NewRecorder()
	New(loaded, imageDir).ServeHTTP(without, httptest.NewRequest("GET", "/", nil))
	if strings.Contains(without.Body.String(), "photo-button") {
		t.Fatal("photo button shown without a model")
	}
	modelDir := t.TempDir()
	writeModelDir(t, modelDir)
	mux, err := NewWithOptions(loaded, imageDir, Options{ModelDir: modelDir})
	if err != nil {
		t.Fatal(err)
	}
	with := httptest.NewRecorder()
	mux.ServeHTTP(with, httptest.NewRequest("GET", "/", nil))
	body := with.Body.String()
	for _, want := range []string{`id="photo-button"`, `data-model-version="20261116-1"`, `data-model-runtime="/model/ort-1.31.1/"`, `id="photo-status"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("home page misses %s", want)
		}
	}
}

func TestServeModelFiles(t *testing.T) {
	loaded, imageDir := photoTestCatalog(t)
	modelDir := t.TempDir()
	writeModelDir(t, modelDir)
	mux, err := NewWithOptions(loaded, imageDir, Options{ModelDir: modelDir})
	if err != nil {
		t.Fatal(err)
	}
	get := func(path string, headers map[string]string) *httptest.ResponseRecorder {
		request := httptest.NewRequest("GET", path, nil)
		for key, value := range headers {
			request.Header.Set(key, value)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		return response
	}

	plain := get("/model/model.int8.onnx?v=20261116-1", nil)
	if plain.Code != http.StatusOK || plain.Body.String() != "onnx-bytes" || plain.Header().Get("ETag") == "" {
		t.Fatalf("model: %d %q etag %q", plain.Code, plain.Body.String(), plain.Header().Get("ETag"))
	}
	if again := get("/model/model.int8.onnx", map[string]string{"If-None-Match": plain.Header().Get("ETag")}); again.Code != http.StatusNotModified {
		t.Fatalf("If-None-Match: want 304, got %d", again.Code)
	}

	packed := get("/model/ort-1.31.1/ort-wasm-simd-threaded.wasm", map[string]string{"Accept-Encoding": "br, gzip"})
	if packed.Header().Get("Content-Encoding") != "gzip" || packed.Header().Get("Content-Type") != "application/wasm" || packed.Header().Get("X-Decoded-Length") != "4000" {
		t.Fatalf("gzip variant: headers %v", packed.Header())
	}
	reader, err := gzip.NewReader(packed.Body)
	if err != nil {
		t.Fatal(err)
	}
	if unpacked, _ := io.ReadAll(reader); len(unpacked) != 4000 {
		t.Fatalf("gzip body has %d bytes", len(unpacked))
	}
	if raw := get("/model/ort-1.31.1/ort-wasm-simd-threaded.wasm", nil); raw.Header().Get("Content-Encoding") != "" || raw.Body.Len() != 4000 {
		t.Fatal("client without gzip must get the plain file")
	}
	if js := get("/model/ort-1.31.1/ort.wasm.min.mjs", nil); !strings.HasPrefix(js.Header().Get("Content-Type"), "text/javascript") {
		t.Fatalf("mjs content type %q", js.Header().Get("Content-Type"))
	}
	for _, path := range []string{"/model/../species.jsonl", "/model/%2e%2e/species.jsonl", "/model/ort-1.31.1/ort-wasm-simd-threaded.wasm.gz", "/model/missing.bin", "/model/ort-1.31.1"} {
		if response := get(path, nil); response.Code == http.StatusOK {
			t.Fatalf("%s must not be served, got 200", path)
		}
	}
}

func TestSpeciesCardsKeepOrderAndSkipUnknown(t *testing.T) {
	loaded, imageDir := photoTestCatalog(t)
	response := httptest.NewRecorder()
	New(loaded, imageDir).ServeHTTP(response, httptest.NewRequest("GET", "/api/species?slugs=Fliegenpilz,Unbekannt,Steinpilz", nil))
	var body struct {
		Results []searchResult `json:"results"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Results) != 2 || body.Results[0].Slug != "Fliegenpilz" || body.Results[1].Slug != "Steinpilz" || body.Results[1].Image == "" {
		t.Fatalf("unexpected cards %+v", body.Results)
	}
}
