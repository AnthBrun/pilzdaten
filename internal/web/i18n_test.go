package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"pilzdaten/internal/catalog"
	"pilzdaten/internal/i18n"
)

func translatedServer(t *testing.T) (*catalog.Catalog, *i18n.Bundle, func(path string, header ...string) *httptest.ResponseRecorder) {
	t.Helper()
	dir := t.TempDir()
	speciesPath := filepath.Join(dir, "species.jsonl")
	indexPath := filepath.Join(dir, "index.jsonl")
	if err := os.WriteFile(speciesPath, []byte(`{"slug":"Steinpilz","name_trivial":"Steinpilz, Herrenpilz","name_wissenschaftlich":"BOLETUS EDULIS","merkmale":{"hut":"braun, polsterförmig","vorkommen":"Fichtenwald"},"bilddateien":[]}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(indexPath, []byte(`{"slug":"Steinpilz","name":"Steinpilz","wissenschaftlich":"BOLETUS EDULIS","gattung":"Röhrlinge"}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := catalog.Load(speciesPath, indexPath)
	if err != nil {
		t.Fatal(err)
	}
	bundle := i18n.NewBundle(
		map[string]i18n.Dict{"en": {
			"Alles wischen":            "Clear all",
			"{count} passende Art":     "{count} matching species",
			"Essbarkeit nicht erfasst": "Edibility not recorded",
			"Pilzdatenbank":            "Mushroom database",
		}},
		map[string]i18n.Dict{"en": {
			"Steinpilz, Herrenpilz": "Penny bun, Cep",
			"Hut":                   "Cap",
			"braun, polsterförmig":  "brown, cushion-shaped",
			"Röhrlinge":             "Boletes",
		}},
	)
	handler := NewWithTranslations(loaded, dir, bundle)
	get := func(path string, header ...string) *httptest.ResponseRecorder {
		request := httptest.NewRequest("GET", path, nil)
		for index := 0; index+1 < len(header); index += 2 {
			request.Header.Set(header[index], header[index+1])
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	return loaded, bundle, get
}

func TestPagesFollowRequestedLanguage(t *testing.T) {
	_, _, get := translatedServer(t)

	german := get("/").Body.String()
	if !strings.Contains(german, "Alles wischen") || !strings.Contains(german, `<html lang="de">`) || !strings.Contains(german, ">Steinpilz<") {
		t.Error("default page should be German")
	}

	byHeader := get("/", "Accept-Language", "en-GB,en;q=0.9").Body.String()
	for _, expected := range []string{`<html lang="en">`, "Clear all", "1 matching species", ">Penny bun<", `<option value="en" lang="en" selected>English</option>`, "Maschinell übersetzt"} {
		if !strings.Contains(byHeader, expected) {
			t.Errorf("English page does not contain %q", expected)
		}
	}
	if !strings.Contains(byHeader, `id="i18n">{"`) || !strings.Contains(byHeader, `"Alles wischen":"Clear all"`) {
		t.Error("English page should embed the UI dictionary for the JS modules")
	}

	switched := get("/?lang=en", "Accept-Language", "de")
	if cookie := switched.Header().Get("Set-Cookie"); !strings.HasPrefix(cookie, "lang=en") {
		t.Errorf("?lang=en should remember the language, Set-Cookie = %q", cookie)
	}
	if !strings.Contains(switched.Body.String(), "Clear all") {
		t.Error("?lang=en should render English")
	}
	if byCookie := get("/", "Cookie", "lang=en", "Accept-Language", "de").Body.String(); !strings.Contains(byCookie, "Clear all") {
		t.Error("lang cookie should win over Accept-Language")
	}
	if unknown := get("/?lang=xx").Body.String(); !strings.Contains(unknown, "Alles wischen") {
		t.Error("unknown language should fall back to German")
	}
}

func TestDetailTranslatesSpeciesContent(t *testing.T) {
	_, _, get := translatedServer(t)
	body := get("/art/Steinpilz", "Cookie", "lang=en").Body.String()
	for _, expected := range []string{
		`<h1 class="detail-title">Penny bun</h1>`,
		`<span class="synonym">Cep</span>`, `<span class="synonym">Steinpilz</span>`, // deutsche Namen bleiben als Alias
		"<dt>Cap</dt><dd>brown, cushion-shaped</dd>",
		"<dt>Vorkommen</dt><dd>Fichtenwald</dd>", // ohne Übersetzung bleibt es deutsch
		`<p class="eyebrow">Boletes</p>`,
		"Edibility not recorded",
		"BOLETUS EDULIS",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("English detail page does not contain %q", expected)
		}
	}
}

func TestSearchAPIAndOfflinePackAreTranslated(t *testing.T) {
	_, _, get := translatedServer(t)
	var result searchResponse
	if err := json.Unmarshal(get("/api/search?q=penny", "Cookie", "lang=en").Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 || result.Results[0].Name != "Penny bun" || result.Results[0].EdibilityLabel != "Edibility not recorded" {
		t.Errorf("English search for a translated name = %+v", result)
	}
	// Deutsche Namen bleiben in jeder Sprache suchbar.
	if err := json.Unmarshal(get("/api/search?q=herrenpilz", "Cookie", "lang=en").Body.Bytes(), &result); err != nil || result.Count != 1 {
		t.Errorf("German name should still be found in English, got %+v, %v", result, err)
	}

	german := get("/api/offline")
	english := get("/api/offline", "Cookie", "lang=en")
	if german.Header().Get("ETag") == english.Header().Get("ETag") {
		t.Error("offline packs of different languages need different ETags")
	}
	var pack offlinePackage
	if err := json.Unmarshal(english.Body.Bytes(), &pack); err != nil {
		t.Fatal(err)
	}
	species := pack.Species[0]
	if pack.Lang != "en" || species.Name != "Penny bun" || !strings.Contains(species.Names, "Herrenpilz") || species.Genus != "Boletes" {
		t.Errorf("English offline pack = lang %q, %+v", pack.Lang, species)
	}
	if !slices.Contains(species.Traits, [2]string{"Cap", "brown, cushion-shaped"}) {
		t.Errorf("offline traits are not translated: %v", species.Traits)
	}
}

func TestTextsCollectsEverythingTheUIAndDataUse(t *testing.T) {
	loaded, _, _ := translatedServer(t)
	ui, content := Texts(loaded)
	for _, expected := range []string{
		"Alles wischen",          // Template
		"{count} passende Arten", // Template mit Platzhalter (tf)
		"Hutfarbe",               // JS (config.js)
		"{part} wählen",          // JS mit Platzhalter
		"Giftig",                 // Go, i18n.N
		"Seite konnte nicht gerendert werden",
	} {
		if !slices.Contains(ui, expected) {
			t.Errorf("ui texts miss %q", expected)
		}
	}
	// "Hut" steht auch in der Oberfläche (config.js) und wird nur einmal übersetzt.
	if !slices.Contains(ui, "Hut") || slices.Contains(content, "Hut") {
		t.Error(`"Hut" should be collected once, as UI text`)
	}
	for _, expected := range []string{"Steinpilz, Herrenpilz", "Röhrlinge", "braun, polsterförmig", "Fichtenwald", "Vorkommen"} {
		if !slices.Contains(content, expected) {
			t.Errorf("content texts miss %q", expected)
		}
	}
	for _, text := range append(ui, content...) {
		if !translatable(text) {
			t.Errorf("untranslatable text %q collected", text)
		}
	}
}
