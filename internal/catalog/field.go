package catalog

import (
	"strconv"
	"strings"
	"unicode"
)

// Merkmale, die im Wald sofort beobachtbar sind (Saison, Baumpartner), stehen
// in der Quelle nur als Freitext im Feld "vorkommen". Sie werden beim Laden in
// virtuelle Filter-Tags übersetzt, damit Filter, Zähler und Ausgrauen ohne
// Sonderfälle funktionieren.

var monthNames = []string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}

// MonthTag liefert den Filter-Tag für einen Monat (1–12).
func MonthTag(month int) string {
	return "monat_" + strconv.Itoa(month)
}

// seasonWords bildet Jahreszeit- und Monatswörter auf einen Monatsbereich ab.
var seasonWords = map[string][2]int{
	"frühjahr": {3, 5}, "frühling": {3, 5},
	"frühsommer": {6, 6}, "hochsommer": {7, 8}, "spätsommer": {8, 9}, "sommer": {6, 8},
	"frühherbst": {9, 9}, "spätherbst": {11, 11}, "herbst": {9, 11},
	"frühwinter": {12, 12}, "spätwinter": {2, 2}, "winter": {12, 2},
	"januar": {1, 1}, "februar": {2, 2}, "märz": {3, 3}, "april": {4, 4}, "mai": {5, 5}, "juni": {6, 6},
	"juli": {7, 7}, "august": {8, 8}, "september": {9, 9}, "oktober": {10, 10}, "november": {11, 11}, "dezember": {12, 12},
}

type textWord struct {
	word       string
	start, end int
}

func splitWords(text string) []textWord {
	words := make([]textWord, 0)
	start := -1
	for index, r := range text {
		isLetter := unicode.IsLetter(r)
		if isLetter && start < 0 {
			start = index
		}
		if !isLetter && start >= 0 {
			words = append(words, textWord{word: text[start:index], start: start, end: index})
			start = -1
		}
	}
	if start >= 0 {
		words = append(words, textWord{word: text[start:], start: start, end: len(text)})
	}
	return words
}

// seasonMonths liest aus einem Vorkommen-Text die Monate, in denen die Art
// erscheint. "Frühsommer bis Spätherbst" wird zum Bereich Juni–November.
// Ohne erkennbare Angabe gilt die Art als ganzjährig, damit sie nicht durch
// fehlende Daten aus den Treffern fällt.
func seasonMonths(text string) [13]bool {
	var months [13]bool
	lower := strings.ToLower(text)
	if strings.Contains(lower, "ganzjährig") || strings.Contains(lower, "ganzes jahr") {
		return allMonths()
	}
	type seasonToken struct {
		from, to   int
		start, end int
	}
	tokens := make([]seasonToken, 0)
	for _, word := range splitWords(lower) {
		if span, ok := seasonWords[word.word]; ok {
			tokens = append(tokens, seasonToken{from: span[0], to: span[1], start: word.start, end: word.end})
		}
	}
	if len(tokens) == 0 {
		return allMonths()
	}
	for index, token := range tokens {
		from := token.from
		if index > 0 {
			between := lower[tokens[index-1].end:token.start]
			if strings.Contains(between, "bis") || strings.ContainsAny(between, "-–") {
				from = tokens[index-1].from
			}
		}
		for month := from; ; month = month%12 + 1 {
			months[month] = true
			if month == token.to {
				break
			}
		}
	}
	return months
}

func allMonths() [13]bool {
	var months [13]bool
	for month := 1; month <= 12; month++ {
		months[month] = true
	}
	return months
}

type forestTag struct {
	Tag, Label string
	terms      []string
}

// forestTags werden am Wortanfang erkannt ("Eichenwald") oder hinter einer
// bekannten Baumart-Vorsilbe ("Stieleiche"), damit "weichen" oder
// "Bereichen" nicht als Eiche zählen.
var forestTags = []forestTag{
	{Tag: "wald_laub", Label: "Laubwald", terms: []string{"laubwald", "laubwäld", "laubbaum", "laubbäum", "laubholz", "laubgehölz", "mischwald", "mischwäld", "buche", "eiche", "birke", "pappel", "erle", "hasel", "linde", "ahorn", "esche", "kastanie"}},
	{Tag: "wald_nadel", Label: "Nadelwald", terms: []string{"nadelwald", "nadelwäld", "nadelbaum", "nadelbäum", "nadelholz", "nadelgehölz", "nadelstreu", "mischwald", "mischwäld", "fichte", "kiefer", "tanne", "lärche", "douglasie"}},
	{Tag: "baum_buche", Label: "Buche", terms: []string{"buche"}},
	{Tag: "baum_eiche", Label: "Eiche", terms: []string{"eiche"}},
	{Tag: "baum_birke", Label: "Birke", terms: []string{"birke"}},
	{Tag: "baum_pappel", Label: "Pappel", terms: []string{"pappel"}},
	{Tag: "baum_fichte", Label: "Fichte", terms: []string{"fichte"}},
	{Tag: "baum_kiefer", Label: "Kiefer", terms: []string{"kiefer"}},
	{Tag: "baum_laerche", Label: "Lärche", terms: []string{"lärche"}},
}

func forestTagsFor(text string) []string {
	words := splitWords(strings.ToLower(text))
	tags := make([]string, 0)
	for _, candidate := range forestTags {
		if containsForestTerm(words, candidate.terms) {
			tags = append(tags, candidate.Tag)
		}
	}
	return tags
}

// treeQualifiers sind Vorsilben echter Baumarten-Komposita. "Hainbuche" fehlt
// bewusst, sie ist botanisch keine Buche.
var treeQualifiers = map[string]bool{
	"rot": true, "blut": true, "stiel": true, "trauben": true, "zerr": true, "kork": true, "stein": true, "flaum": true,
	"wald": true, "schwarz": true, "berg": true, "latschen": true, "weiß": true, "edel": true, "sand": true,
	"moor": true, "hänge": true, "sitka": true, "silber": true, "zitter": true, "grau": true, "ess": true, "ross": true,
	"winter": true, "sommer": true, "spitz": true, "feld": true,
}

func containsForestTerm(words []textWord, terms []string) bool {
	for _, word := range words {
		for _, term := range terms {
			index := strings.Index(word.word, term)
			if index == 0 || index > 0 && treeQualifiers[word.word[:index]] {
				return true
			}
		}
	}
	return false
}

// hatTags kennzeichnen Pilze mit Hut und Fruchtschicht auf der Unterseite.
var hatTags = map[string]bool{"lamellen": true, "roehren": true, "stachel": true, "leisten": true}

// addFieldTags ergänzt Saison- und Wald-Tags aus dem Vorkommen-Text sowie den
// abgeleiteten Gruppen-Tag "hutpilz".
func addFieldTags(tags []string, traits map[string]any) []string {
	for _, tag := range tags {
		if hatTags[tag] {
			tags = appendUniqueTag(tags, "hutpilz")
			break
		}
	}
	text, _ := traits["vorkommen"].(string)
	months := seasonMonths(text)
	for month := 1; month <= 12; month++ {
		if months[month] {
			tags = appendUniqueTag(tags, MonthTag(month))
		}
	}
	for _, tag := range forestTagsFor(text) {
		tags = appendUniqueTag(tags, tag)
	}
	return tags
}

// addFieldFacets hängt die Gruppen "saison" und "wald" an die Facetten an
// und zählt, wie viele Arten jeden Tag tragen.
func (c *Catalog) addFieldFacets() {
	counts := make(map[string]int)
	for _, filterData := range c.filterData {
		for _, tag := range filterData.Tags {
			counts[tag]++
		}
	}
	season := FacetGroup{Group: "saison", Title: "Fundmonat"}
	for month := 1; month <= 12; month++ {
		tag := MonthTag(month)
		season.Options = append(season.Options, FacetOption{Tag: tag, Label: monthNames[month-1], Count: counts[tag]})
	}
	forest := FacetGroup{Group: "wald", Title: "Wald & Baumpartner"}
	for _, candidate := range forestTags {
		forest.Options = append(forest.Options, FacetOption{Tag: candidate.Tag, Label: candidate.Label, Count: counts[candidate.Tag]})
	}
	// Hauptmerkmal wie auf 123pilzsuche.de: Zuerst wird die Pilzgruppe gewählt.
	group := FacetGroup{Group: "gruppe", Title: "Was für ein Pilz?"}
	titles := make(map[string]string)
	for _, facetGroup := range c.facets.Groups {
		for _, option := range facetGroup.Options {
			if option.Title != "" {
				titles[option.Tag] = option.Title
			}
		}
	}
	for _, tag := range []string{"hutpilz", "kugel", "hut_koralle", "konsistenz_gallertartig", "f", "sonstige_gattung"} {
		label := sourceLabels[tag]
		if tag == "hutpilz" {
			label = "Hut & Stiel"
		}
		group.Options = append(group.Options, FacetOption{Tag: tag, Label: label, Count: counts[tag], Title: titles[tag]})
	}
	c.facets.Groups = append(c.facets.Groups, season, forest, group)
}

// sourcePriority liest das Quellfeld "priorität" (1 = bekannte, häufig
// gesuchte Art, 3 = Spezialfall). Fehlende Werte zählen als unwichtigste.
func sourcePriority(traits map[string]any) int {
	var value string
	switch raw := traits["priorität"].(type) {
	case string:
		value = raw
	case float64:
		value = strconv.Itoa(int(raw))
	}
	priority, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || priority < 1 {
		return 4
	}
	return priority
}

// commonness schätzt aus Vorkommen-Text und Quell-Tags, wie oft man einer
// Art begegnet: positiv für "häufig"/"verbreitet", negativ für "selten" und
// Rote-Liste-Arten.
func commonness(traits map[string]any, tags []string) int {
	text, _ := traits["vorkommen"].(string)
	lower := strings.ToLower(text)
	score := 0
	switch {
	case strings.Contains(lower, "sehr häufig"):
		score += 3
	case strings.Contains(lower, "häufig") && !strings.Contains(lower, "nicht häufig"):
		score += 2
	}
	if strings.Contains(lower, "verbreitet") && !strings.Contains(lower, "nicht verbreitet") {
		score++
	}
	for _, word := range splitWords(lower) {
		switch word.word {
		case "selten":
			score--
		case "rl":
			score--
		}
	}
	if strings.Contains(lower, "sehr selten") || strings.Contains(lower, "extrem selten") {
		score -= 2
	}
	for _, tag := range tags {
		if tag == "nicht_selten" {
			score++
		}
	}
	return score
}

// photoBucket fasst die Bildanzahl grob zusammen; ab 40 Fotos gilt eine Art
// als gut dokumentiert, damit Sammelbegriffe mit Hunderten Bildern nicht
// automatisch vorne stehen.
func photoBucket(count int) int {
	if count > 40 {
		count = 40
	}
	return count / 10
}

func countPhotos(paths []string) int {
	count := 0
	for _, path := range paths {
		if strings.HasSuffix(strings.ToLower(path), ".jpg") {
			count++
		}
	}
	return count
}

// speciesRelevanceLess sortiert so, dass man im Wald zuerst die Arten sieht,
// denen man wahrscheinlich begegnet: wichtige vor seltenen, gut bebilderte
// vor kaum dokumentierten.
func speciesRelevanceLess(left, right *Species) bool {
	if left.Priority != right.Priority {
		return left.Priority < right.Priority
	}
	if left.Commonness != right.Commonness {
		return left.Commonness > right.Commonness
	}
	if leftPhotos, rightPhotos := photoBucket(left.PhotoCount), photoBucket(right.PhotoCount); leftPhotos != rightPhotos {
		return leftPhotos > rightPhotos
	}
	return speciesNameLess(left, right)
}
