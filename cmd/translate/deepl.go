package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// deepL ist ein schlanker Client für die DeepL-API v2 (translate, glossaries).
type deepL struct {
	key  string
	base string
	http *http.Client
}

func newDeepL(key string) *deepL {
	base := "https://api.deepl.com"
	if strings.HasSuffix(key, ":fx") { // Schlüssel der kostenlosen API
		base = "https://api-free.deepl.com"
	}
	if override := os.Getenv("DEEPL_API_URL"); override != "" {
		base = strings.TrimRight(override, "/")
	}
	return &deepL{key: key, base: base, http: &http.Client{Timeout: 2 * time.Minute}}
}

// translationContext hilft DeepL bei Mehrdeutigem (Hut, Stiel, Ring …); wird nicht berechnet.
const translationContext = "Texte einer App zur Pilzbestimmung: Merkmale von Pilzen (Hut, Lamellen, Röhren, Stiel, Fleisch, Sporen), Fundorte, Essbarkeit und Giftigkeit."

// translate übersetzt ein Paket deutscher Texte; die Reihenfolge bleibt erhalten.
func (d *deepL) translate(ctx context.Context, texts []string, lang, glossaryID string) ([]string, error) {
	xml := hasPlaceholder(texts[0])
	request := map[string]any{
		"text":                texts,
		"source_lang":         "DE",
		"target_lang":         targetLang(lang),
		"formality":           "prefer_less", // die App duzt
		"preserve_formatting": true,
		"context":             translationContext,
	}
	if glossaryID != "" {
		request["glossary_id"] = glossaryID
	}
	if xml {
		encoded := make([]string, len(texts))
		for index, text := range texts {
			encoded[index] = toXML(text)
		}
		request["text"] = encoded
		request["tag_handling"] = "xml"
		request["ignore_tags"] = []string{"x"}
	}
	var response struct {
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}
	if err := d.call(ctx, http.MethodPost, "/v2/translate", request, &response); err != nil {
		return nil, err
	}
	if len(response.Translations) != len(texts) {
		return nil, fmt.Errorf("DeepL lieferte %d statt %d Übersetzungen", len(response.Translations), len(texts))
	}
	result := make([]string, len(texts))
	for index, translation := range response.Translations {
		result[index] = translation.Text
		if xml {
			result[index] = fromXML(translation.Text)
		}
	}
	return result, nil
}

// glossaryFor legt aus einer TSV-Datei ein Glossar an (fehlt die Datei: keins).
func (d *deepL) glossaryFor(ctx context.Context, lang, path string) (string, error) {
	entries, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var response struct {
		ID string `json:"glossary_id"`
	}
	request := map[string]any{
		"name": "pilzdaten-" + lang, "source_lang": "de", "target_lang": glossaryLang(lang),
		"entries": strings.TrimSpace(string(entries)) + "\n", "entries_format": "tsv",
	}
	if err := d.call(ctx, http.MethodPost, "/v2/glossaries", request, &response); err != nil {
		return "", err
	}
	fmt.Printf("%s: Glossar %s angelegt\n", lang, response.ID)
	return response.ID, nil
}

// deleteGlossary räumt das Glossar nach dem Lauf wieder ab.
func (d *deepL) deleteGlossary(id string) {
	if d == nil || id == "" {
		return
	}
	_ = d.call(context.Background(), http.MethodDelete, "/v2/glossaries/"+id, nil, nil)
}

// call schickt eine Anfrage; bei Überlast (429, 503, 529) wird mit wachsender Pause wiederholt.
func (d *deepL) call(ctx context.Context, method, path string, body, result any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	for attempt := 1; ; attempt++ {
		request, err := http.NewRequestWithContext(ctx, method, d.base+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		request.Header.Set("Authorization", "DeepL-Auth-Key "+d.key)
		if body != nil {
			request.Header.Set("Content-Type", "application/json")
		}
		response, err := d.http.Do(request)
		if err != nil {
			if attempt < 5 && ctx.Err() == nil {
				sleep(ctx, attempt)
				continue
			}
			return err
		}
		data, _ := io.ReadAll(response.Body)
		response.Body.Close()
		switch {
		case response.StatusCode < 300:
			if result == nil || len(data) == 0 {
				return nil
			}
			return json.Unmarshal(data, result)
		case (response.StatusCode == 429 || response.StatusCode == 503 || response.StatusCode == 529) && attempt < 8:
			sleep(ctx, attempt)
		case response.StatusCode == 456:
			return errors.New("DeepL-Kontingent aufgebraucht (456)")
		case response.StatusCode == 403:
			return errors.New("DeepL lehnt den Schlüssel ab (403)")
		default:
			return fmt.Errorf("DeepL %s %s: %d %s", method, path, response.StatusCode, strings.TrimSpace(string(data)))
		}
	}
}

func sleep(ctx context.Context, attempt int) {
	select {
	case <-ctx.Done():
	case <-time.After(time.Duration(attempt*attempt) * time.Second):
	}
}

// targetLang: DeepL verlangt bei manchen Sprachen eine Variante.
func targetLang(lang string) string {
	switch lang {
	case "en":
		return "EN-GB"
	case "pt":
		return "PT-PT"
	case "zh":
		return "ZH-HANS"
	default:
		return strings.ToUpper(lang)
	}
}

// glossaryLang: Glossare nehmen die Sprache ohne Variante.
func glossaryLang(lang string) string {
	return strings.ToLower(lang)
}

// --- Platzhalter: {name} geht als <x id="name"/> an DeepL und bleibt so unübersetzt.

var (
	placeholderPattern = regexp.MustCompile(`\{([a-z]+)\}`)
	xmlTagPattern      = regexp.MustCompile(`<x id="([a-z]+)"\s*(?:/>|></x>)`)
)

func hasPlaceholder(text string) bool {
	return placeholderPattern.MatchString(text)
}

func toXML(text string) string {
	return placeholderPattern.ReplaceAllString(html.EscapeString(text), `<x id="$1"/>`)
}

func fromXML(text string) string {
	return html.UnescapeString(xmlTagPattern.ReplaceAllString(text, "{$1}"))
}
