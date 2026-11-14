package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTranslateFallsBackToGerman(t *testing.T) {
	bundle := NewBundle(map[string]Dict{"en": {"Hut": "Cap"}}, map[string]Dict{"en": {"Fichtenwald": "Spruce forest"}})
	for _, test := range []struct{ lang, text, want string }{
		{"en", "Hut", "Cap"},
		{"en", "Fichtenwald", "Spruce forest"},
		{"en", "Stiel", "Stiel"},
		{"de", "Hut", "Hut"},
		{"fr", "Hut", "Hut"},
	} {
		if got := bundle.T(test.lang, test.text); got != test.want {
			t.Errorf("T(%q, %q) = %q, want %q", test.lang, test.text, got, test.want)
		}
	}
	var none *Bundle
	if got := none.T("en", "Hut"); got != "Hut" {
		t.Errorf("nil bundle T = %q", got)
	}
}

func TestMatchPicksBestOfferedLanguage(t *testing.T) {
	bundle := NewBundle(map[string]Dict{"en": {}, "fr": {}}, nil)
	for header, want := range map[string]string{
		"":                           "de",
		"en-US,en;q=0.9":             "en",
		"it-IT,it;q=0.9,fr;q=0.5":    "fr",
		"de-AT,de;q=0.9,en;q=0.8":    "de",
		"es;q=0.9,en;q=0.1,fr;q=0.6": "fr",
		"pl, cs;q=0.8":               "de",
		"en;q=0, fr;q=0.2":           "fr",
		"EN-gb":                      "en",
	} {
		if got := bundle.Match(header); got != want {
			t.Errorf("Match(%q) = %q, want %q", header, got, want)
		}
	}
}

func TestFormatReplacesPlaceholders(t *testing.T) {
	if got := Format("Seite {page} von {pages}", "page", 2, "pages", 7); got != "Seite 2 von 7" {
		t.Errorf("Format = %q", got)
	}
}

func TestLoadReadsLanguageFolders(t *testing.T) {
	dir := t.TempDir()
	if err := WriteDict(filepath.Join(dir, "en", "ui.json"), Dict{"Hut": "Cap"}); err != nil {
		t.Fatal(err)
	}
	if err := WriteDict(filepath.Join(dir, "fr", "content.json"), Dict{"Wald": "Forêt"}); err != nil {
		t.Fatal(err)
	}
	// leere Sprachordner, ungültige Namen und Dateien werden ignoriert
	for _, name := range []string{"cs", "glossary", "README"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	bundle, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	codes := []string{}
	for _, lang := range bundle.Languages() {
		codes = append(codes, lang.Code)
	}
	if len(codes) != 3 || codes[0] != "de" || codes[1] != "en" || codes[2] != "fr" {
		t.Fatalf("languages = %v, want [de en fr]", codes)
	}
	if bundle.T("en", "Hut") != "Cap" || bundle.T("fr", "Wald") != "Forêt" {
		t.Error("loaded translations are not used")
	}
	if missing, err := Load(filepath.Join(dir, "gibt-es-nicht")); err != nil || len(missing.Languages()) != 1 {
		t.Errorf("missing dir should give German only, got %v, %v", missing.Languages(), err)
	}
}
