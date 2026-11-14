// Package i18n hält die Übersetzungen der Pilzdatenbank.
//
// Deutsch ist die Quellsprache. Jede weitere Sprache liegt als Ordner unter
// data/i18n/<code>/ mit zwei Wörterbüchern „deutscher Text → Übersetzung“:
//
//	ui.json       Oberfläche, Filter-Labels, feste Hinweise (wird auch ins JS eingebettet)
//	content.json  Artdaten: Namen, Gattungen, Merkmalsnamen und -texte
//
// Fehlt eine Übersetzung, bleibt der deutsche Text stehen. Die Dateien
// erzeugt cmd/translate (DeepL); Schlüssel ist immer der deutsche Text, so
// fällt eine geänderte Quelle automatisch aus dem Wörterbuch und wird neu übersetzt.
package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Source ist die Quellsprache aller Texte.
const Source = "de"

// Dict ordnet deutschen Texten ihre Übersetzung zu.
type Dict map[string]string

// Lang beschreibt eine angebotene Sprache.
type Lang struct {
	Code string // ISO 639-1, z. B. "en"
	Name string // Name in der eigenen Sprache, z. B. "English"
}

// Bundle enthält alle geladenen Sprachen. Der Nullwert (oder nil) kennt nur Deutsch.
type Bundle struct {
	langs   []Lang
	ui      map[string]Dict
	content map[string]Dict
}

// Load liest alle Sprachordner unter dir. Fehlt dir, gibt es nur Deutsch.
func Load(dir string) (*Bundle, error) {
	bundle := &Bundle{ui: map[string]Dict{}, content: map[string]Dict{}}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return bundle.finish(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("Übersetzungen %q lesen: %w", dir, err)
	}
	for _, entry := range entries {
		code := entry.Name()
		if !entry.IsDir() || !validCode(code) || code == Source {
			continue
		}
		ui, err := ReadDict(filepath.Join(dir, code, "ui.json"))
		if err != nil {
			return nil, err
		}
		content, err := ReadDict(filepath.Join(dir, code, "content.json"))
		if err != nil {
			return nil, err
		}
		if len(ui) == 0 && len(content) == 0 {
			continue
		}
		bundle.ui[code], bundle.content[code] = ui, content
	}
	return bundle.finish(), nil
}

// NewBundle baut ein Bundle aus Wörterbüchern im Speicher (für Tests).
func NewBundle(ui, content map[string]Dict) *Bundle {
	bundle := &Bundle{ui: map[string]Dict{}, content: map[string]Dict{}}
	for code, dict := range ui {
		bundle.ui[code] = dict
	}
	for code, dict := range content {
		bundle.content[code] = dict
		if bundle.ui[code] == nil {
			bundle.ui[code] = Dict{}
		}
	}
	return bundle.finish()
}

func (b *Bundle) finish() *Bundle {
	b.langs = []Lang{{Code: Source, Name: LanguageName(Source)}}
	codes := make([]string, 0, len(b.ui))
	for code := range b.ui {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		b.langs = append(b.langs, Lang{Code: code, Name: LanguageName(code)})
	}
	return b
}

// Languages liefert Deutsch und alle geladenen Sprachen.
func (b *Bundle) Languages() []Lang {
	if b == nil {
		return []Lang{{Code: Source, Name: LanguageName(Source)}}
	}
	return b.langs
}

// Has meldet, ob die Sprache angeboten wird.
func (b *Bundle) Has(code string) bool {
	for _, lang := range b.Languages() {
		if lang.Code == code {
			return true
		}
	}
	return false
}

// T übersetzt einen deutschen Text; ohne Übersetzung bleibt er deutsch.
func (b *Bundle) T(code, text string) string {
	if b == nil || code == Source || text == "" {
		return text
	}
	if translated, ok := b.ui[code][text]; ok && translated != "" {
		return translated
	}
	if translated, ok := b.content[code][text]; ok && translated != "" {
		return translated
	}
	return text
}

// UI liefert das Oberflächen-Wörterbuch einer Sprache (fürs JS).
func (b *Bundle) UI(code string) Dict {
	if b == nil || b.ui[code] == nil {
		return Dict{}
	}
	return b.ui[code]
}

// Match wählt aus einem Accept-Language-Header die beste angebotene Sprache.
func (b *Bundle) Match(acceptLanguage string) string {
	type candidate struct {
		code    string
		quality float64
		order   int
	}
	candidates := make([]candidate, 0)
	for index, part := range strings.Split(acceptLanguage, ",") {
		tag, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		code := strings.ToLower(strings.TrimSpace(tag))
		code, _, _ = strings.Cut(code, "-")
		quality := 1.0
		if q, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
			if _, err := fmt.Sscanf(q, "%g", &quality); err != nil {
				quality = 0
			}
		}
		if code != "" && quality > 0 {
			candidates = append(candidates, candidate{code, quality, index})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].quality > candidates[j].quality })
	for _, candidate := range candidates {
		if b.Has(candidate.code) {
			return candidate.code
		}
	}
	return Source
}

// Format ersetzt {name}-Platzhalter durch die Werte aus pairs ("name", wert, …).
func Format(text string, pairs ...any) string {
	for index := 0; index+1 < len(pairs); index += 2 {
		text = strings.ReplaceAll(text, "{"+fmt.Sprint(pairs[index])+"}", fmt.Sprint(pairs[index+1]))
	}
	return text
}

// --- Registrierung fester Texte aus Go-Code

var (
	registryMu sync.Mutex
	registry   = map[string]struct{}{}
)

// N markiert einen festen deutschen Text aus Go-Code zur Übersetzung und gibt
// ihn unverändert zurück. Aufrufe gehören in Paketvariablen, damit
// cmd/translate den Text schon beim Programmstart kennt.
func N(text string) string {
	registryMu.Lock()
	registry[text] = struct{}{}
	registryMu.Unlock()
	return text
}

// Registered liefert alle mit N markierten Texte.
func Registered() []string {
	registryMu.Lock()
	defer registryMu.Unlock()
	texts := make([]string, 0, len(registry))
	for text := range registry {
		texts = append(texts, text)
	}
	sort.Strings(texts)
	return texts
}

// --- Dateien

// ReadDict liest ein Wörterbuch; eine fehlende Datei ist ein leeres Wörterbuch.
func ReadDict(path string) (Dict, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Dict{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Übersetzung %q lesen: %w", path, err)
	}
	dict := Dict{}
	if err := json.Unmarshal(data, &dict); err != nil {
		return nil, fmt.Errorf("Übersetzung %q lesen: %w", path, err)
	}
	return dict, nil
}

// WriteDict schreibt ein Wörterbuch sortiert und atomar (erst Temp-Datei, dann umbenennen).
func WriteDict(path string, dict Dict) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(dict, "", "  ") // encoding/json sortiert Map-Schlüssel
	if err != nil {
		return err
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(temp, path)
}

func validCode(code string) bool {
	if len(code) < 2 || len(code) > 3 {
		return false
	}
	for _, r := range code {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

// LanguageName liefert den Namen einer Sprache in ihr selbst.
func LanguageName(code string) string {
	if name, ok := languageNames[code]; ok {
		return name
	}
	return strings.ToUpper(code)
}

var languageNames = map[string]string{
	"de": "Deutsch", "en": "English", "fr": "Français", "it": "Italiano", "es": "Español",
	"nl": "Nederlands", "pl": "Polski", "cs": "Čeština", "sk": "Slovenčina", "sl": "Slovenščina",
	"hu": "Magyar", "da": "Dansk", "sv": "Svenska", "nb": "Norsk", "fi": "Suomi", "pt": "Português",
	"ro": "Română", "hr": "Hrvatski", "bg": "Български", "el": "Ελληνικά", "ru": "Русский",
	"uk": "Українська", "tr": "Türkçe", "ja": "日本語", "zh": "中文", "ko": "한국어",
	"et": "Eesti", "lv": "Latviešu", "lt": "Lietuvių", "qa": "Pseudo",
}
