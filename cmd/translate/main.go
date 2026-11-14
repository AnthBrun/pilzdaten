// translate übersetzt alle Texte der Pilzdatenbank mit DeepL nach data/i18n/<sprache>/.
//
//	go run ./cmd/translate -lang en,fr -dry-run   # zählt nur, was zu übersetzen wäre
//	go run ./cmd/translate -lang en               # übersetzt alles Fehlende
//	go run ./cmd/translate -lang en -limit 200    # höchstens 200 Texte je Datei (zum Testen)
//	go run ./cmd/translate -lang qa -pseudo       # Pseudo-Sprache ohne DeepL, zeigt unübersetzte Stellen
//
// Geschickt werden nur Texte, die im Wörterbuch noch fehlen; Texte, die es in
// den Daten nicht mehr gibt, fliegen raus. Nach jedem Paket wird gespeichert,
// ein abgebrochener Lauf macht beim nächsten Start dort weiter.
//
// Der Schlüssel kommt aus DEEPL_API_KEY oder aus der .env im aktuellen Ordner.
// Optional: Glossar data/i18n/glossary/<sprache>.tsv (deutsch<TAB>übersetzung je Zeile).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"pilzdaten/internal/catalog"
	"pilzdaten/internal/i18n"
	"pilzdaten/internal/web"
)

func main() {
	langs := flag.String("lang", "", "Zielsprachen, kommagetrennt (z. B. en,fr)")
	scope := flag.String("scope", "all", "ui, content oder all")
	dryRun := flag.Bool("dry-run", false, "nur zählen, nichts senden")
	limit := flag.Int("limit", 0, "höchstens so viele neue Texte je Datei (0 = alle)")
	pseudo := flag.Bool("pseudo", false, "Pseudo-Übersetzung ‹…› statt DeepL (nur für -lang qa)")
	dataDir := flag.String("data", envOr("DATA_DIR", "data"), "Datenordner")
	flag.Parse()
	log.SetFlags(0)

	codes := splitList(*langs)
	if len(codes) == 0 {
		log.Fatal("-lang fehlt, z. B. -lang en,fr")
	}
	if *pseudo && (len(codes) != 1 || codes[0] != "qa") {
		log.Fatal("-pseudo nur mit -lang qa, damit keine echte Sprache überschrieben wird")
	}
	data, err := catalog.LoadDir(*dataDir)
	if err != nil {
		log.Fatalf("Daten laden: %v", err)
	}
	ui, content := web.Texts(data)
	files := map[string][]string{"ui": ui, "content": content}
	if *scope != "all" {
		if _, ok := files[*scope]; !ok {
			log.Fatalf("-scope muss ui, content oder all sein")
		}
		files = map[string][]string{*scope: files[*scope]}
	}

	var client *deepL
	if !*dryRun && !*pseudo {
		key := apiKey()
		if key == "" {
			log.Fatal("DEEPL_API_KEY fehlt (Umgebungsvariable oder .env)")
		}
		client = newDeepL(key)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	totalChars := 0
	for _, code := range codes {
		var glossaryID string
		if client != nil {
			glossaryID, err = client.glossaryFor(ctx, code, filepath.Join(*dataDir, "i18n", "glossary", code+".tsv"))
			if err != nil {
				log.Fatalf("%s: Glossar: %v", code, err)
			}
		}
		for _, name := range []string{"ui", "content"} {
			texts, ok := files[name]
			if !ok {
				continue
			}
			path := filepath.Join(*dataDir, "i18n", code, name+".json")
			chars, err := translateFile(ctx, client, code, glossaryID, path, texts, *limit, *dryRun, *pseudo)
			totalChars += chars
			if err != nil {
				client.deleteGlossary(glossaryID)
				log.Fatalf("%s/%s: %v", code, name, err)
			}
		}
		client.deleteGlossary(glossaryID)
	}
	if *dryRun {
		fmt.Printf("Summe: %d Zeichen\n", totalChars)
	}
}

// translateFile bringt eine Wörterbuchdatei auf den Stand von texts und
// liefert die Zeichenzahl der (zu) übersetzenden Texte.
func translateFile(ctx context.Context, client *deepL, code, glossaryID, path string, texts []string, limit int, dryRun, pseudo bool) (int, error) {
	dict, err := i18n.ReadDict(path)
	if err != nil {
		return 0, err
	}
	wanted := make(map[string]struct{}, len(texts))
	missing := make([]string, 0)
	for _, text := range texts {
		wanted[text] = struct{}{}
		if _, done := dict[text]; !done {
			missing = append(missing, text)
		}
	}
	removed := 0
	for text := range dict {
		if _, ok := wanted[text]; !ok {
			delete(dict, text)
			removed++
		}
	}
	if limit > 0 && len(missing) > limit {
		missing = missing[:limit]
	}
	chars := 0
	for _, text := range missing {
		chars += utf8.RuneCountInString(text)
	}
	label := fmt.Sprintf("%s/%s", code, filepath.Base(path))
	fmt.Printf("%s: %d Texte gesamt, %d vorhanden, %d neu (%d Zeichen), %d veraltet\n", label, len(texts), len(texts)-len(missing), len(missing), chars, removed)
	if dryRun {
		return chars, nil
	}
	if pseudo {
		for _, text := range missing {
			dict[text] = "‹" + text + "›"
		}
		return chars, i18n.WriteDict(path, dict)
	}
	if removed > 0 || len(missing) == 0 {
		if err := i18n.WriteDict(path, dict); err != nil {
			return chars, err
		}
	}
	done, doneChars := 0, 0
	for _, batch := range batches(missing) {
		if err := ctx.Err(); err != nil {
			return chars, fmt.Errorf("abgebrochen nach %d Texten (gespeichert, der nächste Lauf macht weiter)", done)
		}
		translated, err := client.translate(ctx, batch, code, glossaryID)
		if err != nil {
			return chars, fmt.Errorf("nach %d von %d Texten: %w (bisheriges ist gespeichert)", done, len(missing), err)
		}
		for index, text := range batch {
			dict[text] = translated[index]
			doneChars += utf8.RuneCountInString(text)
		}
		done += len(batch)
		if err := i18n.WriteDict(path, dict); err != nil {
			return chars, err
		}
		fmt.Printf("  %s: %d/%d Texte, %d/%d Zeichen\n", label, done, len(missing), doneChars, chars)
	}
	return chars, nil
}

// batches teilt Texte in DeepL-Pakete: höchstens 50 Texte und ~100 KB je Anfrage.
// Texte mit Platzhaltern ({name}) kommen in eigene Pakete, weil sie als XML gehen.
func batches(texts []string) [][]string {
	const maxTexts, maxBytes = 50, 100_000
	result := make([][]string, 0)
	for _, placeholders := range []bool{false, true} {
		current, size := []string{}, 0
		for _, text := range texts {
			if hasPlaceholder(text) != placeholders {
				continue
			}
			if len(current) > 0 && (len(current) == maxTexts || size+len(text) > maxBytes) {
				result = append(result, current)
				current, size = []string{}, 0
			}
			current = append(current, text)
			size += len(text)
		}
		if len(current) > 0 {
			result = append(result, current)
		}
	}
	return result
}

func apiKey() string {
	if key := strings.TrimSpace(os.Getenv("DEEPL_API_KEY")); key != "" {
		return key
	}
	env, err := os.ReadFile(".env")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(env), "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "DEEPL_API_KEY="); ok {
			return strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return ""
}

func splitList(value string) []string {
	items := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		if item = strings.ToLower(strings.TrimSpace(item)); item != "" && item != i18n.Source {
			items = append(items, item)
		}
	}
	return items
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
