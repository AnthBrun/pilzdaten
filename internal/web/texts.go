package web

import (
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"pilzdaten/internal/catalog"
	"pilzdaten/internal/i18n"
)

// Texts sammelt alle deutschen Texte, die cmd/translate übersetzen muss:
//
//	ui       Templates ({{t "…"}}, {{tf "…"}}), JS (t("…")), mit i18n.N
//	         markierte Go-Texte sowie Titel und Labels der Filtergruppen
//	content  Artdaten: Trivialnamen, Gattungen, Merkmalsnamen und -texte
//
// Die Texte sind genau die Schlüssel, unter denen die Seiten nachschlagen.
func Texts(data *catalog.Catalog) (ui, content []string) {
	uiSet := map[string]struct{}{}
	add := func(set map[string]struct{}, text string) {
		if translatable(text) {
			set[text] = struct{}{}
		}
	}

	for _, text := range embeddedTexts() {
		add(uiSet, text)
	}
	for _, text := range i18n.Registered() {
		add(uiSet, text)
	}
	for _, group := range data.Facets().Groups {
		add(uiSet, group.Title)
		for _, option := range group.Options {
			add(uiSet, option.Label)
			add(uiSet, option.Title)
			if group.Group == "spore" {
				add(uiSet, sporeLabel(option.Label))
			}
		}
	}

	contentSet := map[string]struct{}{}
	for _, species := range data.All() {
		add(contentSet, species.NameTrivial)
		add(contentSet, species.Gattung)
		main, supplemental := speciesTraits(species)
		for _, value := range append(main, supplemental...) {
			add(contentSet, value.Name)
			add(contentSet, value.Value)
		}
	}
	// Was schon in der Oberfläche steht, braucht kein zweites Mal übersetzt zu werden.
	for text := range uiSet {
		delete(contentSet, text)
	}
	return sortedKeys(uiSet), sortedKeys(contentSet)
}

var (
	templateTextPattern = regexp.MustCompile(`(?:\{\{-?|\()\s*tf?\s+("(?:[^"\\]|\\.)*")`)
	jsTextPattern       = regexp.MustCompile(`\bt\(\s*("(?:[^"\\]|\\.)*")`)
)

// embeddedTexts liest die Übersetzungsaufrufe aus den eingebetteten Templates und JS-Modulen.
func embeddedTexts() []string {
	texts := make([]string, 0)
	collect := func(pattern *regexp.Regexp, glob string) {
		paths, _ := fs.Glob(files, glob)
		for _, path := range paths {
			source, err := fs.ReadFile(files, path)
			if err != nil {
				continue
			}
			for _, match := range pattern.FindAllSubmatch(source, -1) {
				if text, err := strconv.Unquote(string(match[1])); err == nil {
					texts = append(texts, text)
				}
			}
		}
	}
	collect(templateTextPattern, "templates/*.html")
	collect(jsTextPattern, "static/js/*.js")
	return texts
}

// translatable: leere Texte und reine Zahlen/Zeichen gehen nicht an DeepL.
func translatable(text string) bool {
	return strings.IndexFunc(text, unicode.IsLetter) >= 0
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
