package web

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pilzdaten/internal/catalog"
)

func TestDetailRendersTraitsImagesAndRelatedSpecies(t *testing.T) {
	dir := t.TempDir()
	speciesPath := filepath.Join(dir, "species.jsonl")
	indexPath := filepath.Join(dir, "index.jsonl")
	imageDir := filepath.Join(dir, "images")
	if err := os.MkdirAll(imageDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "pilz.jpg"), []byte("test-image"), 0o600); err != nil {
		t.Fatal(err)
	}
	data := `{"slug":"Steinpilz","url":"https://example.test/steinpilz","name_trivial":"Steinpilz","name_wissenschaftlich":"BOLETUS EDULIS","synonyme":["Herrenpilz"],"merkmale":{"hut":"braun","geruch":"mild"},"bilddateien":["pilz.jpg"],"verwechslungen":["Gallenroehrling"]}
{"slug":"Gallenroehrling","name_trivial":"Gallenröhrling","name_wissenschaftlich":"Tylopilus felleus","merkmale":{},"bilddateien":[],"verwechslungen":[]}
`
	index := `{"slug":"Steinpilz","name":"Steinpilz","wissenschaftlich":"BOLETUS EDULIS","gattung":"Röhrlinge"}
{"slug":"Gallenroehrling","name":"Gallenröhrling","wissenschaftlich":"Tylopilus felleus","gattung":"Röhrlinge"}
`
	if err := os.WriteFile(speciesPath, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(indexPath, []byte(index), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := catalog.Load(speciesPath, indexPath)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(loaded, imageDir)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/art/Steinpilz", nil))
	if response.Code != 200 {
		t.Fatalf("detail status = %d, want 200", response.Code)
	}
	for _, expected := range []string{"BOLETUS EDULIS", "Herrenpilz", "Merkmale", "braun", "Bilder", "/media/pilz.jpg", `loading="lazy"`, "Gallenröhrling", "Essbarkeit nicht erfasst"} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("detail page does not contain %q", expected)
		}
	}

	imageResponse := httptest.NewRecorder()
	handler.ServeHTTP(imageResponse, httptest.NewRequest("GET", "/media/pilz.jpg", nil))
	if imageResponse.Code != 200 || imageResponse.Body.String() != "test-image" {
		t.Fatalf("image response = %d, %q", imageResponse.Code, imageResponse.Body.String())
	}

	notFound := httptest.NewRecorder()
	handler.ServeHTTP(notFound, httptest.NewRequest("GET", "/art/unknown", nil))
	if notFound.Code != 404 {
		t.Fatalf("unknown species status = %d, want 404", notFound.Code)
	}
}
