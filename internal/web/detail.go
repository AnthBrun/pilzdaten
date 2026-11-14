package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"unicode"

	"pilzdaten/internal/catalog"
	"pilzdaten/internal/i18n"
)

// detailData füllt templates/detail.html. Gegenstück für den Offline-Betrieb
// ist renderSpecies in static/js/offline.js – Änderungen bitte in beiden.
type detailData struct {
	DisplayName        string
	Aliases            []string
	Species            *catalog.Species
	Traits             []trait
	SupplementalTraits []trait
	Images             []image
	Gallery            []image
	Related            []*catalog.Species
	DangerousRelated   []*catalog.Species
}

type trait struct {
	Name  string
	Value string
}

type image struct {
	URL    string
	Full   string // großes Original für die Lightbox, leer wenn nicht vorhanden
	Alt    string
	Number int // 1-basierte Position unter allen Fotos der Art (Hauptbild = 1)
}

func (s *Server) detail(w http.ResponseWriter, r *http.Request) {
	species, ok := s.catalog.Get(r.PathValue("slug"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	lang := s.requestLang(w, r)
	localNames := s.t(lang, species.NameTrivial)
	data := detailData{Species: species, DisplayName: displayName(localNames)}
	// Übersetzte Namen zuerst, dahinter die deutschen Originalnamen und die Synonyme.
	seen := map[string]bool{data.DisplayName: true}
	for _, names := range []string{localNames, species.NameTrivial} {
		for _, alias := range strings.Split(names, ",") {
			if alias = strings.TrimSpace(alias); alias != "" && !seen[alias] {
				seen[alias] = true
				data.Aliases = append(data.Aliases, alias)
			}
		}
	}
	data.Aliases = append(data.Aliases, species.Synonyme...)
	main, supplemental := speciesTraits(species)
	data.Traits, data.SupplementalTraits = s.localTraits(lang, main), s.localTraits(lang, supplemental)
	data.Images = speciesPhotos(species, s.catalog.Original)
	for index := range data.Images {
		data.Images[index].Alt = data.DisplayName
	}
	if len(data.Images) > 1 {
		data.Gallery = data.Images[1:]
	}
	for _, slug := range species.Verwechslungen {
		if related, found := s.catalog.Get(slug); found {
			data.Related = append(data.Related, related)
			if related.EdibilityWarning {
				data.DangerousRelated = append(data.DangerousRelated, related)
			}
		}
	}
	// Giftige Doppelgänger zuerst: Sie sind im Wald die wichtigste Information.
	sort.SliceStable(data.Related, func(i, j int) bool {
		return dangerRank(data.Related[i]) < dangerRank(data.Related[j])
	})
	s.render(w, lang, "detail.html", data)
}

func dangerRank(species *catalog.Species) int {
	switch species.EdibilityClass {
	case "deadly":
		return 0
	case "poisonous":
		return 1
	default:
		return 2
	}
}

// speciesPhotos liefert die echten Fotos einer Art, das bevorzugte Bild zuerst.
// original liefert zum Bildpfad das große Original oder "".
func speciesPhotos(species *catalog.Species, original func(string) string) []image {
	seen := make(map[string]bool)
	photos := make([]image, 0)
	addPhoto := func(path string) {
		if path == "" || !isPhoto(path) || seen[path] {
			return
		}
		seen[path] = true
		photo := image{URL: imageURL(path), Alt: species.NameTrivial, Number: len(photos) + 1}
		if full := original(path); full != "" {
			photo.Full = originalURL(full)
		}
		photos = append(photos, photo)
	}
	addPhoto(species.BestImage)
	for _, imagePath := range species.Bilddateien {
		addPhoto(imagePath)
	}
	return photos
}

// --- Merkmale

// speciesTraits ordnet die Quellmerkmale wie auf der Artseite: die
// Erkennungsmerkmale zuerst, Kommentare und Sonderfelder getrennt dahinter.
func speciesTraits(species *catalog.Species) (main, supplemental []trait) {
	allTraits := make([]trait, 0, len(species.Merkmale))
	for key, value := range species.Merkmale {
		if isHiddenDetailTrait(key) {
			continue
		}
		if rendered := displayValue(value); rendered != "" {
			allTraits = append(allTraits, trait{Name: label(key), Value: rendered})
		}
	}
	sort.Slice(allTraits, func(i, j int) bool {
		left, right := allTraits[i], allTraits[j]
		leftOrder, rightOrder := detailTraitOrder(left.Name), detailTraitOrder(right.Name)
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		return strings.ToLower(left.Name) < strings.ToLower(right.Name)
	})
	for _, value := range allTraits {
		if detailTraitOrder(value.Name) < supplementalOrder {
			main = append(main, value)
		} else {
			supplemental = append(supplemental, value)
		}
	}
	return main, supplemental
}

// localTraits übersetzt Namen und Texte der Merkmale (Reihenfolge bleibt).
func (s *Server) localTraits(lang string, traits []trait) []trait {
	if lang == i18n.Source {
		return traits
	}
	local := make([]trait, len(traits))
	for index, value := range traits {
		local[index] = trait{Name: s.t(lang, value.Name), Value: s.t(lang, value.Value)}
	}
	return local
}

// Merkmale ab dieser Reihenfolge landen unter "Weitere Angaben".
const supplementalOrder = 9

// detailTraitOrder sortiert Merkmale grob von oben nach unten am Pilz.
func detailTraitOrder(name string) int {
	name = strings.ToLower(strings.TrimSpace(name))
	switch {
	case containsTraitTerm(name, "hut", "fruchtkörper"):
		return 0
	case containsTraitTerm(name, "lamell", "röhr", "rohr", "por", "stachel", "leisten", "unterseite", "fruchtschicht"):
		return 1
	case containsTraitTerm(name, "stiel", "ring", "knoll", "wurzel"):
		return 2
	case containsTraitTerm(name, "fleisch", "milch"):
		return 3
	case containsTraitTerm(name, "geruch", "geschmack"):
		return 4
	case containsTraitTerm(name, "spore"):
		return 5
	case containsTraitTerm(name, "vorkommen", "standort", "substrat", "habitat"):
		return 6
	case containsTraitTerm(name, "essbarkeit", "gift", "verzehr", "unverträg"):
		return 7
	case containsTraitTerm(name, "gattung", "taxonomie"):
		return 8
	default:
		return supplementalOrder
	}
}

// Priorität ist ein internes Sortierfeld, Verwechslungen stehen als eigene Sektion da.
func isHiddenDetailTrait(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "priorität", "prioritaet":
		return true
	}
	return strings.Contains(name, "verwechslung")
}

func containsTraitTerm(value string, terms ...string) bool {
	words := strings.FieldsFunc(value, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, term := range terms {
		for _, word := range words {
			if strings.HasPrefix(word, term) {
				return true
			}
		}
	}
	return false
}

// displayValue macht aus einem Merkmalswert der Quelle (Text, Liste, Objekt) Anzeigetext.
func displayValue(value any) string {
	switch value := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	case []any:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			if part := displayValue(item); part != "" {
				parts = append(parts, part)
			}
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprint(value)
		}
		return string(encoded)
	default:
		return fmt.Sprint(value)
	}
}
