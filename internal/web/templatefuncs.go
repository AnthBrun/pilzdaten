package web

import (
	"html/template"
	"net/url"
	"strings"
	"unicode"
)

var templateFuncs = template.FuncMap{
	"imageURL":     imageURL,
	"firstImage":   firstImage,
	"displayName":  displayName,
	"sporeLabel":   sporeLabel,
	"sensoryGroup": sensoryGroup,
	"shortMonth":   shortMonth,
}

// imageURL macht aus einem Bildpfad der Quelle ("daten/details/<Slug>-Dateien/image004.jpg")
// die URL unter /media/.
func imageURL(path string) string {
	parts := strings.Split(strings.TrimLeft(path, "/"), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return "/media/" + strings.Join(parts, "/")
}

// isPhoto unterscheidet echte Fotos (.jpg) von den kleinen .gif-Markern der Quelle.
func isPhoto(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".jpg")
}

// firstImage liefert das erste echte Foto einer Art; die Bildlisten der Quelle
// beginnen mit 15×15-gif-Markern. Ohne Foto wird der erste Eintrag genommen.
func firstImage(paths []string) string {
	for _, path := range paths {
		if isPhoto(path) {
			return path
		}
	}
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}

// displayName gibt den ersten Trivialnamen eines Komma-getrennten Namensstrings zurück.
func displayName(nameTrivial string) string {
	trimmed := strings.TrimSpace(nameTrivial)
	if i := strings.Index(trimmed, ","); i >= 0 {
		return strings.TrimSpace(trimmed[:i])
	}
	return trimmed
}

// label macht aus einem Schlüssel einen Anzeigetext ("sporen_farbe" → "Sporen farbe").
func label(value string) string {
	value = strings.ReplaceAll(value, "_", " ")
	runes := []rune(value)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return string(runes)
}

// shortMonth kürzt Monatsnamen für die Chip-Leiste ("September" → "Sep").
func shortMonth(name string) string {
	runes := []rune(name)
	if len(runes) <= 4 {
		return name
	}
	return string(runes[:3])
}

func sporeLabel(value string) string {
	return strings.TrimSpace(strings.TrimPrefix(value, "Sporen "))
}

// sensoryGroup ordnet Geruchs-/Geschmacks-Tags der Chip-Liste zu, wie die
// Kategorien "Geruch" und "Geschmack" auf 123pilzsuche.de. Die Quelle führt
// beide in einer Facette ("geruch"); Geschmack sind nur diese fünf Tags.
func sensoryGroup(tag string) string {
	switch tag {
	case "mild", "bitter", "scharf", "pilziger_geschmack", "sonstiger_geschmack":
		return "geschmack"
	default:
		return "geruch"
	}
}
