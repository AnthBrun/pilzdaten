package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesSpeciesIndexAndPreservesVariableTraits(t *testing.T) {
	dir := t.TempDir()
	speciesPath := filepath.Join(dir, "species.jsonl")
	indexPath := filepath.Join(dir, "index.jsonl")

	if err := os.WriteFile(speciesPath, []byte(`{"slug":"Boletus","name_trivial":"Steinpilz","name_wissenschaftlich":"BOLETUS EDULIS","synonyme":["Herrenpilz"],"merkmale":{"hut":"braun","ring":false},"bilddateien":["bilder/a.jpg"],"verwechslungen":["Gallenroehrling"]}
{"slug":"ListenSeite","name_trivial":"Artenliste","merkmale":{}}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(indexPath, []byte(`{"slug":"Boletus","name":"Steinpilz","wissenschaftlich":"BOLETUS EDULIS","gattung":"Röhrlinge"}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	catalog, err := Load(speciesPath, indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Count() != 1 {
		t.Fatalf("Count() = %d, want 1", catalog.Count())
	}
	species, ok := catalog.Get("Boletus")
	if !ok {
		t.Fatal("indexed species was not loaded")
	}
	if species.Merkmale["hut"] != "braun" || species.Gattung != "Röhrlinge" {
		t.Fatalf("species fields were not preserved: %#v", species)
	}
	if _, ok := catalog.Get("ListenSeite"); ok {
		t.Fatal("non-species page was loaded")
	}
}
