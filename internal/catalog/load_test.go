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
	available := loaded.AvailableTags("", []string{"braun"}, 5, 12)
	if available["lamellen"] != 3 || available["gut"] != 2 {
		t.Fatalf("available tags with braun and diameter 5–12 = %#v, want lamellen=3 and gut=2", available)
	}
	if _, included := available["braun"]; included {
		t.Fatalf("selected tag braun should not be included in available tags: %#v", available)
	}
	available = loaded.AvailableTags("B Pilz", []string{"braun"}, 5, 12)
	if available["lamellen"] != 1 || available["gut"] != 0 {
		t.Fatalf("available tags with query B Pilz = %#v, want lamellen=1 and no gut", available)
	}
}

func TestLeistenCategoryIsAddedAsFilterTag(t *testing.T) {
	dir := t.TempDir()
	speciesPath := filepath.Join(dir, "species.jsonl")
	indexPath := filepath.Join(dir, "index.jsonl")
	filtersPath := filepath.Join(dir, "filter_tags.jsonl")
	if err := os.WriteFile(speciesPath, []byte(`{"slug":"Leistenpilz","name_trivial":"Leistenpilz","merkmale":{"leisten":"Leisten"}}
{"slug":"Mischform","name_trivial":"Mischform","merkmale":{"lamellen/leisten":"Lamellen und Leisten"}}
{"slug":"Lamellenpilz","name_trivial":"Lamellenpilz","merkmale":{"lamellen":"Lamellen"}}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(indexPath, []byte(`{"slug":"Leistenpilz","name":"Leistenpilz"}
{"slug":"Mischform","name":"Mischform"}
{"slug":"Lamellenpilz","name":"Lamellenpilz"}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filtersPath, []byte(`{"slug":"Leistenpilz","tags":["lamellen"]}
{"slug":"Mischform","tags":["lamellen"]}
{"slug":"Lamellenpilz","tags":["lamellen"]}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadWithFilters(speciesPath, indexPath, filtersPath, "")
	if err != nil {
		t.Fatal(err)
	}
	results, count, _ := loaded.Filter("", []string{"leisten"}, 1, 25, "", 1, 50)
	if count != 2 || len(results) != 2 || results[0].Slug != "Leistenpilz" || results[1].Slug != "Mischform" {
		t.Fatalf("Leisten filter results = %#v (count %d), want Leistenpilz and Mischform", results, count)
	}
	available := loaded.AvailableTags("", nil, 1, 25)
	if available["leisten"] != 2 {
		t.Fatalf("available Leisten count = %d, want 2", available["leisten"])
	}
}

func TestFacetLabelsAreShortenedForCapAndColorChange(t *testing.T) {
	catalog := &Catalog{facets: Facets{Groups: []FacetGroup{
		{Group: "hut", Options: []FacetOption{
			{Tag: "braun", Label: "Hut braun"},
			{Tag: "hut_gerieft", Label: "Hut gerieft"},
		}},
		{Group: "farbwechsel", Options: []FacetOption{
			{Tag: "farbwechsel_kein", Label: "Kein Farbwechsel"},
			{Tag: "farbwechsel_grau", Label: "Wird grau"},
		}},
		{Group: "stiel", Options: []FacetOption{{Tag: "stiel_braun", Label: "Brauner Stiel"}}},
	}}}

	catalog.normalizeFacetLabels()
	got := []string{
		catalog.facets.Groups[0].Options[0].Label,
		catalog.facets.Groups[0].Options[1].Label,
		catalog.facets.Groups[1].Options[0].Label,
		catalog.facets.Groups[1].Options[1].Label,
		catalog.facets.Groups[2].Options[0].Label,
	}
	want := []string{"braun", "gerieft", "Keiner", "grau", "Brauner Stiel"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("normalized label %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoadWithAssetsUsesPreferredPhotoAndEdibilityMetadata(t *testing.T) {
	dir := t.TempDir()
	paths := map[string]string{
		"species": filepath.Join(dir, "species.jsonl"),
		"index":   filepath.Join(dir, "index.jsonl"),
		"filters": filepath.Join(dir, "filter_tags.jsonl"),
		"facets":  filepath.Join(dir, "facets.json"),
		"images":  filepath.Join(dir, "best_images.jsonl"),
		"icons":   filepath.Join(dir, "pictogram_map.json"),
	}
	fixtures := map[string]string{
		"species": `{"slug":"A","name_trivial":"Art A","bilddateien":["old.jpg"]}` + "\n",
		"index":   `{"slug":"A","name":"Art A"}` + "\n",
		"filters": `{"slug":"A","tags":["hervorragend"]}` + "\n",
		"facets":  `{"min_diameter_cm":1,"max_diameter_cm":25,"groups":[{"group":"hut","title":"Hut","options":[{"tag":"braun","label":"Hut braun"}]},{"group":"essbarkeit","title":"Essbarkeit","options":[{"tag":"hervorragend","label":"Hervorragend"}]}]}`,
		"images":  `{"slug":"A","image":"large.jpg"}` + "\n",
		"icons":   `{"braun":{"title":"Hut &szlig;braun.","img":"braun","img_active":"braun_ja"}}`,
	}
	for key, path := range paths {
		if err := os.WriteFile(path, []byte(fixtures[key]), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	loaded, err := LoadWithAssets(paths["species"], paths["index"], paths["filters"], paths["facets"], paths["images"], paths["icons"])
	if err != nil {
		t.Fatal(err)
	}
	species, ok := loaded.Get("A")
	if !ok {
		t.Fatal("species A was not loaded")
	}
	if species.BestImage != "large.jpg" || species.EdibilityLabel != "Hervorragend essbar" || species.EdibilityClass != "excellent" {
		t.Fatalf("loaded presentation data = image %q, edibility %q (%s)", species.BestImage, species.EdibilityLabel, species.EdibilityClass)
	}
	facets := loaded.Facets()
	if len(facets.Groups) != 2 || facets.Groups[0].Group != "essbarkeit" {
		t.Fatalf("first facet group = %#v, want essbarkeit promoted first", facets.Groups)
	}
	option := facets.Groups[1].Options[0]
	if option.Title != "Hut ßbraun." || option.Image != "braun" || option.ActiveImage != "braun_ja" {
		t.Fatalf("pictogram metadata = %#v", option)
	}
}
