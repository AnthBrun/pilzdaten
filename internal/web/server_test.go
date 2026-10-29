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
	data := `{"slug":"Steinpilz","url":"https://example.test/steinpilz","name_trivial":"Steinpilz","name_wissenschaftlich":"BOLETUS EDULIS","synonyme":["Herrenpilz"],"merkmale":{"priorität":"2","kommentar":"Hinweis","vorkommen":"Wald","sporenpulverfarbe":"braun","fleisch":"weiß","stiel":"lang","lamellen":"hell","hut":"braun","geschmack":"bitter","geruch":"pilzig","relativer speisewert":"2","tipp":"Nur als Beispiel","lustige anmerkung":"Anekdote","verwechslungsgefahr":"Gallenröhrling"},"bilddateien":["pilz.jpg"],"verwechslungen":["Gallenroehrling"]}
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
	for _, expected := range []string{"BOLETUS EDULIS", "Herrenpilz", "Merkmale", "Geruch", "Geschmack", "bitter", "braun", "Bilder", "/media/pilz.jpg", `loading="lazy"`, "Gallenröhrling", "Essbarkeit nicht erfasst"} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("detail page does not contain %q", expected)
		}
	}
	if strings.Contains(response.Body.String(), "Priorität") {
		t.Error("detail page should not show the source priority field")
	}
	if strings.Contains(response.Body.String(), "<dt>Verwechslungsgefahr</dt>") {
		t.Error("detail page should not repeat the textual confusion field")
	}
	if !strings.Contains(response.Body.String(), `<h2 id="related-heading">Verwechslungspartner</h2>`) {
		t.Error("detail page should keep the related-species section with images")
	}
	body := response.Body.String()
	supplementalPosition := strings.Index(body, `<summary><span>Weitere Angaben</span>`)
	if supplementalPosition < 0 {
		t.Fatal("detail page does not provide a collapsed section for supplemental source fields")
	}
	for _, label := range []string{"Kommentar", "Relativer speisewert", "Tipp", "Lustige anmerkung"} {
		position := strings.Index(body, "<dt>"+label+"</dt>")
		if position <= supplementalPosition {
			t.Errorf("supplemental field %q should be inside the collapsed additional-data section", label)
		}
	}
	orderedTraits := []string{"<dt>Hut</dt>", "<dt>Lamellen</dt>", "<dt>Stiel</dt>", "<dt>Fleisch</dt>", "<dt>Geruch</dt>", "<dt>Geschmack</dt>", "<dt>Sporenpulverfarbe</dt>", "<dt>Vorkommen</dt>", "<dt>Kommentar</dt>"}
	lastPosition := -1
	for _, expected := range orderedTraits {
		position := strings.Index(response.Body.String(), expected)
		if position <= lastPosition {
			t.Errorf("trait %q appears at %d, want it after position %d", expected, position, lastPosition)
		}
		lastPosition = position
	}
	if strings.Contains(response.Body.String(), "← Alle Arten") {
		t.Error("detail page still shows the confusing header link")
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

func TestHomeHidesBottomAlphabetAndRetainsLetterFiltering(t *testing.T) {
	dir := t.TempDir()
	speciesPath := filepath.Join(dir, "species.jsonl")
	indexPath := filepath.Join(dir, "index.jsonl")
	if err := os.WriteFile(speciesPath, []byte(`{"slug":"UmlautPilz","name_trivial":"Ästiger Pilz"}
{"slug":"ZahlenPilz","name_trivial":"32-sporiger Pilz"}
{"slug":"SonderPilz","name_trivial":"!Sonderpilz"}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(indexPath, []byte(`{"slug":"UmlautPilz","name":"Ästiger Pilz"}
{"slug":"ZahlenPilz","name":"32-sporiger Pilz"}
{"slug":"SonderPilz","name":"!Sonderpilz"}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := catalog.Load(speciesPath, indexPath)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(loaded, dir)

	for _, test := range []struct {
		path string
		name string
	}{
		{path: "/?letter=A", name: "Ästiger Pilz"},
		{path: "/?letter=Z", name: "32-sporiger Pilz"},
		{path: "/?letter=%23", name: "!Sonderpilz"},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", test.path, nil))
		if response.Code != 200 || !strings.Contains(response.Body.String(), test.name) {
			t.Errorf("GET %s = %d; body does not contain %q", test.path, response.Code, test.name)
		}
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
	if strings.Contains(response.Body.String(), "ARTENVERZEICHNIS") || strings.Contains(response.Body.String(), `class="alphabet"`) {
		t.Error("home page should not render the redundant bottom alphabet")
	}
	body := response.Body.String()
	safetyPosition := strings.Index(body, `class="safety-spoiler"`)
	legendPosition := strings.Index(body, `class="edibility-legend"`)
	resultsPosition := strings.Index(body, `id="species-results"`)
	firstCardPosition := strings.Index(body, `class="species-card`)
	if safetyPosition < 0 || resultsPosition < safetyPosition || !strings.Contains(body, "zu 100&nbsp;% sicher") {
		t.Error("home page should show the safety spoiler and its key warning near the beginning")
	}
	if legendPosition < 0 || resultsPosition < legendPosition || firstCardPosition >= 0 && legendPosition > firstCardPosition {
		t.Error("edibility legend should appear before the result cards")
	}
	for _, expected := range []string{`viewBox="0 0 240 310"`, `transform="translate(0 -20)"`, `class="substrate substrate-base"`, `data-substrate="erde"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("home page does not contain updated mushroom/ground markup %q", expected)
		}
	}
}

func TestHomeSeparatesSmellAndTasteFacets(t *testing.T) {
	dir := t.TempDir()
	paths := map[string]string{
		"species": `{"slug":"Testpilz","name_trivial":"Testpilz","merkmale":{"geruch":"Anisartig","geschmack":"Bitter"}}` + "\n",
		"index":   `{"slug":"Testpilz","name":"Testpilz"}` + "\n",
		"filters": `{"slug":"Testpilz","tags":["anis","bitter","mild","sueß","pilziger_geschmack"]}` + "\n",
		"facets":  `{"min_diameter_cm":1,"max_diameter_cm":25,"groups":[{"group":"geruch","title":"Geruch & Geschmack","options":[{"tag":"anis","label":"Anis"},{"tag":"bitter","label":"Bitter"},{"tag":"erdig","label":"Erdig"},{"tag":"mild","label":"Mild"},{"tag":"sueß","label":"Süß"},{"tag":"pilziger_geschmack","label":"Pilziger Geschmack"}]}]}`,
	}
	filePaths := make(map[string]string, len(paths))
	for name, contents := range paths {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
		filePaths[name] = path
	}
	loaded, err := catalog.LoadWithFilters(filePaths["species"], filePaths["index"], filePaths["filters"], filePaths["facets"])
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	New(loaded, dir).ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
	if response.Code != 200 {
		t.Fatalf("home status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	smellStart := strings.Index(body, `aria-label="Geruchsmerkmale"`)
	tasteStart := strings.Index(body, `aria-label="Geschmacksmerkmale"`)
	if smellStart < 0 || tasteStart <= smellStart {
		t.Fatal("sensory controls are not rendered in smell and taste groups")
	}
	smell := body[smellStart:tasteStart]
	taste := body[tasteStart:]
	if !strings.Contains(smell, `data-trait-tag="anis"`) || strings.Contains(smell, `data-trait-tag="bitter"`) {
		t.Error("smell-specific facet tags are not grouped correctly")
	}
	if !strings.Contains(taste, `data-trait-tag="bitter"`) || !strings.Contains(taste, `data-trait-tag="pilziger_geschmack"`) {
		t.Error("taste-specific facet tags are not grouped correctly")
	}
	for _, tag := range []string{"sueß", "erdig", "mild"} {
		selector := `data-trait-tag="` + tag + `"`
		if !strings.Contains(smell, selector) || !strings.Contains(taste, selector) {
			t.Errorf("ambiguous sensory tag %q is not shown in both groups", tag)
		}
	}
}

func TestSensoryGroupClassifiesKnownTags(t *testing.T) {
	tests := map[string]string{
		"anis":               "geruch",
		"geruch_pilzig":      "geruch",
		"bitter":             "geschmack",
		"pilziger_geschmack": "geschmack",
		"erdig":              "beide",
		"mild":               "beide",
		"sueß":               "beide",
		"future_unknown_tag": "beide",
	}
	for tag, want := range tests {
		if got := sensoryGroup(tag); got != want {
			t.Errorf("sensoryGroup(%q) = %q, want %q", tag, got, want)
		}
	}
}
