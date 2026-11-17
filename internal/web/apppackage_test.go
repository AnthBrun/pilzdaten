package web

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"pilzdaten/internal/catalog"
)

func TestAppPackageHasAllPhotosAndOriginals(t *testing.T) {
	dir := t.TempDir()
	images := `"bilddateien":["a1.jpg","a2.jpg","a3.jpg","a4.jpg","a5.jpg","a6.jpg","a7.jpg","a8.jpg","a9.jpg","marker.gif"]`
	species := `{"slug":"Steinpilz","name_trivial":"Steinpilz","name_wissenschaftlich":"BOLETUS EDULIS","merkmale":{"hut":"braun"},` + images + `,"verwechslungen":[]}
{"slug":"Ohnebild","name_trivial":"Ohnebild","merkmale":{},"bilddateien":[],"verwechslungen":[]}
`
	index := `{"slug":"Steinpilz","name":"Steinpilz","wissenschaftlich":"BOLETUS EDULIS","gattung":"Röhrlinge"}
{"slug":"Ohnebild","name":"Ohnebild","wissenschaftlich":"","gattung":""}
`
	for name, content := range map[string]string{"species.jsonl": species, "index.jsonl": index, "originals.json": `{"a2.jpg":"gross/a2.jpg"}`} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	data, err := catalog.Load(filepath.Join(dir, "species.jsonl"), filepath.Join(dir, "index.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := data.LoadOriginals(filepath.Join(dir, "originals.json")); err != nil {
		t.Fatal(err)
	}
	raw, err := AppPackage(data, nil, "de")
	if err != nil {
		t.Fatal(err)
	}
	var pkg offlinePackage
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.Version == "" || len(pkg.Species) != 2 {
		t.Fatalf("version %q, %d species", pkg.Version, len(pkg.Species))
	}
	for _, entry := range pkg.Species {
		switch entry.Slug {
		case "Steinpilz":
			if len(entry.Images) != 9 || len(entry.Originals) != 9 {
				t.Errorf("images %d, originals %d, want 9 each", len(entry.Images), len(entry.Originals))
			}
			if entry.Originals[1] != "/original/gross/a2.jpg" || entry.Originals[0] != "" {
				t.Errorf("originals = %v", entry.Originals)
			}
		case "Ohnebild":
			if entry.Originals != nil {
				t.Errorf("species without originals should omit o, got %v", entry.Originals)
			}
		}
	}
}
