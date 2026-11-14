package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"pilzdaten/internal/i18n"
)

// fakeDeepL antwortet wie /v2/translate: "EN:" + Text, Platzhalter-Tags bleiben stehen.
type fakeDeepL struct {
	mu       sync.Mutex
	requests []map[string]any
	failures int // so viele Anfragen zuerst mit 429 beantworten
}

func (f *fakeDeepL) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("Authorization") != "DeepL-Auth-Key test-key" {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if f.failures > 0 {
		f.failures--
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	var request map[string]any
	_ = json.NewDecoder(r.Body).Decode(&request)
	f.requests = append(f.requests, request)
	translations := []map[string]string{}
	for _, text := range request["text"].([]any) {
		translations = append(translations, map[string]string{"text": "EN:" + text.(string)})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"translations": translations})
}

func TestTranslateFileSendsOnlyMissingTextsAndKeepsPlaceholders(t *testing.T) {
	fake := &fakeDeepL{failures: 1}
	server := httptest.NewServer(fake)
	defer server.Close()
	t.Setenv("DEEPL_API_URL", server.URL)
	client := newDeepL("test-key")

	path := filepath.Join(t.TempDir(), "en", "ui.json")
	if err := i18n.WriteDict(path, i18n.Dict{"Hut": "Cap", "Veraltet": "Outdated"}); err != nil {
		t.Fatal(err)
	}
	texts := []string{"Hut", "Stiel", "Seite {page} von {pages}", "Salz & <Pfeffer>"}
	chars, err := translateFile(context.Background(), client, "en", "", path, texts, 0, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if chars != len([]rune("Stiel"))+len([]rune("Seite {page} von {pages}"))+len([]rune("Salz & <Pfeffer>")) {
		t.Errorf("chars = %d", chars)
	}
	dict, err := i18n.ReadDict(path)
	if err != nil {
		t.Fatal(err)
	}
	want := i18n.Dict{
		"Hut":                      "Cap", // vorhanden, nicht neu gesendet
		"Stiel":                    "EN:Stiel",
		"Seite {page} von {pages}": "EN:Seite {page} von {pages}",
		"Salz & <Pfeffer>":         "EN:Salz & <Pfeffer>",
	}
	if len(dict) != len(want) {
		t.Errorf("dict = %v, want %v (veraltete Einträge müssen raus)", dict, want)
	}
	for key, value := range want {
		if dict[key] != value {
			t.Errorf("dict[%q] = %q, want %q", key, dict[key], value)
		}
	}

	if len(fake.requests) != 2 {
		t.Fatalf("want 2 requests (plain + placeholders), got %d", len(fake.requests))
	}
	plain, xml := fake.requests[0], fake.requests[1]
	if plain["target_lang"] != "EN-GB" || plain["source_lang"] != "DE" || plain["tag_handling"] != nil {
		t.Errorf("plain request = %v", plain)
	}
	if xml["tag_handling"] != "xml" || xml["text"].([]any)[0] != `Seite <x id="page"/> von <x id="pages"/>` {
		t.Errorf("placeholder request = %v", xml)
	}

	// Zweiter Lauf: nichts mehr zu tun, keine Anfrage.
	if _, err := translateFile(context.Background(), client, "en", "", path, texts, 0, false, false); err != nil {
		t.Fatal(err)
	}
	if len(fake.requests) != 2 {
		t.Errorf("second run sent %d extra requests", len(fake.requests)-2)
	}
}

func TestBatchesRespectLimitsAndSeparatePlaceholders(t *testing.T) {
	texts := []string{}
	for index := 0; index < 120; index++ {
		texts = append(texts, strings.Repeat("x", 10))
	}
	texts = append(texts, "{count} Arten", strings.Repeat("y", 99_990), strings.Repeat("z", 20))
	result := batches(texts)
	for _, batch := range result {
		size, placeholders := 0, hasPlaceholder(batch[0])
		for _, text := range batch {
			size += len(text)
			if hasPlaceholder(text) != placeholders {
				t.Error("batch mixes plain and placeholder texts")
			}
		}
		if len(batch) > 50 || (size > 100_000 && len(batch) > 1) {
			t.Errorf("batch too large: %d texts, %d bytes", len(batch), size)
		}
	}
	total := 0
	for _, batch := range result {
		total += len(batch)
	}
	if total != len(texts) {
		t.Errorf("batches hold %d texts, want %d", total, len(texts))
	}
}

func TestXMLRoundTrip(t *testing.T) {
	for _, text := range []string{"Seite {page} / {pages}", `„{label}“ & <b>`, "ohne"} {
		if got := fromXML(toXML(text)); got != text {
			t.Errorf("round trip %q -> %q", text, got)
		}
	}
	if got := fromXML(`Page <x id="page"></x> of <x id="pages"/>`); got != "Page {page} of {pages}" {
		t.Errorf("fromXML = %q", got)
	}
}
