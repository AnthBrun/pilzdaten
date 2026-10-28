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
	if matches := catalog.Search("hErReN", 10); len(matches) != 1 || matches[0].Slug != "Boletus" {
		t.Fatalf("case-insensitive synonym search = %#v, want Boletus", matches)
	}
	if matches := catalog.Search("boletus edulis", 10); len(matches) != 1 || matches[0].Slug != "Boletus" {
		t.Fatalf("scientific name search = %#v, want Boletus", matches)
	}
}

func TestFilterRequiresAllTagsAppliesDiameterAndKeepsUnknown(t *testing.T) {
	dir := t.TempDir()
	speciesPath := filepath.Join(dir, "species.jsonl")
	indexPath := filepath.Join(dir, "index.jsonl")
	filtersPath := filepath.Join(dir, "filter_tags.jsonl")
	if err := os.WriteFile(speciesPath, []byte(`{"slug":"A","name_trivial":"A Pilz"}
{"slug":"B","name_trivial":"B Pilz"}
{"slug":"C","name_trivial":"C Pilz"}
{"slug":"D","name_trivial":"D Pilz"}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(indexPath, []byte(`{"slug":"A","name":"A Pilz"}
{"slug":"B","name":"B Pilz"}
{"slug":"C","name":"C Pilz"}
{"slug":"D","name":"D Pilz"}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filtersPath, []byte(`{"slug":"A","tags":["braun"],"hut_durchmesser_min_cm":2,"hut_durchmesser_max_cm":15}
{"slug":"A","tags":["lamellen"],"hut_durchmesser_min_cm":2,"hut_durchmesser_max_cm":15}
{"slug":"A","tags":["gut"],"hut_durchmesser_min_cm":2,"hut_durchmesser_max_cm":15}
{"slug":"B","tags":["braun"],"hut_durchmesser_min_cm":2,"hut_durchmesser_max_cm":15}
{"slug":"B","tags":["lamellen"],"hut_durchmesser_min_cm":2,"hut_durchmesser_max_cm":15}
{"slug":"C","tags":["braun"],"hut_durchmesser_min_cm":13,"hut_durchmesser_max_cm":15}
{"slug":"C","tags":["lamellen"],"hut_durchmesser_min_cm":13,"hut_durchmesser_max_cm":15}
{"slug":"C","tags":["gut"],"hut_durchmesser_min_cm":13,"hut_durchmesser_max_cm":15}
{"slug":"D","tags":["braun"],"hut_durchmesser_min_cm":0,"hut_durchmesser_max_cm":0}
{"slug":"D","tags":["lamellen"],"hut_durchmesser_min_cm":0,"hut_durchmesser_max_cm":0}
{"slug":"D","tags":["gut"],"hut_durchmesser_min_cm":0,"hut_durchmesser_max_cm":0}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadWithFilters(speciesPath, indexPath, filtersPath, "")
	if err != nil {
		t.Fatal(err)
	}
	results, count, pages := loaded.Filter("pilz", []string{"braun", "lamellen", "gut"}, 5, 12, "", 1, 50)
	if count != 2 || pages != 1 || len(results) != 2 || results[0].Slug != "A" || results[1].Slug != "D" {
		t.Fatalf("filtered results = %#v, count %d, pages %d; want A and unknown-size D", results, count, pages)
	}
	results, count, _ = loaded.Filter("", []string{"braun", "gut"}, 5, 12, "", 1, 50)
	if count != 2 || len(results) != 2 || results[0].Slug != "A" || results[1].Slug != "D" {
		t.Fatalf("AND-filter results = %#v, count %d; want A and D with all requested tags", results, count)
	}
	_, count, _ = loaded.Filter("", nil, 1, 25, "", 1, 50)
	if count != 4 {
		t.Fatalf("default diameter range results = %d, want all four species", count)
	}
}
