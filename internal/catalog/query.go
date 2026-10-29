package catalog

import (
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Search returns species matching their common name, scientific name, or any
// synonym. Matching is case-insensitive and results follow catalog order.
func (c *Catalog) Search(query string, limit int) []*Species {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	results := make([]*Species, 0)
	for _, species := range c.all {
		if containsFold(species.NameTrivial, query) || containsFold(species.NameWissenschaftlich, query) || containsAnyFold(species.Synonyme, query) {
			results = append(results, species)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}
	return results
}

// List returns one A-Z letter page and the total number of matching pages.
func (c *Catalog) List(letter string, page, pageSize int) ([]*Species, int) {
	letter = strings.ToUpper(strings.TrimSpace(letter))
	if pageSize < 1 {
		pageSize = 50
	}
	if page < 1 {
		page = 1
	}
	filtered := make([]*Species, 0)
	for _, species := range c.all {
		if initialLetter(species.NameTrivial) == letter {
			filtered = append(filtered, species)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return speciesNameLess(filtered[i], filtered[j]) })
	pages := (len(filtered) + pageSize - 1) / pageSize
	start := (page - 1) * pageSize
	if start >= len(filtered) {
		return []*Species{}, pages
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], pages
}

// Filter returns matching species for a text query, all selected tags, a
// diameter interval, and optionally an initial letter. Page count is returned
// separately so callers can render paginated results.
func (c *Catalog) Filter(query string, tags []string, minCM, maxCM int, letter string, page, pageSize int) ([]*Species, int, int) {
	query = strings.ToLower(strings.TrimSpace(query))
	letter = strings.ToUpper(strings.TrimSpace(letter))
	if pageSize < 1 {
		pageSize = 50
	}
	if page < 1 {
		page = 1
	}
	selected := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if tag = strings.TrimSpace(tag); tag != "" {
			selected[tag] = struct{}{}
		}
	}

	matches := make([]*Species, 0)
	for _, species := range c.all {
		if letter != "" && initialLetter(species.NameTrivial) != letter {
			continue
		}
		filterData := c.filterData[species.Slug]
		if !matchesQueryAndDiameter(species, filterData, query, minCM, maxCM) {
			continue
		}
		present := make(map[string]struct{}, len(filterData.Tags))
		for _, tag := range filterData.Tags {
			present[tag] = struct{}{}
		}
		matchesTags := true
		for tag := range selected {
			if _, ok := present[tag]; !ok {
				matchesTags = false
				break
			}
		}
		if !matchesTags {
			continue
		}
		matches = append(matches, species)
	}
	sort.Slice(matches, func(i, j int) bool { return speciesNameLess(matches[i], matches[j]) })

	total := len(matches)
	pages := (total + pageSize - 1) / pageSize
	start := (page - 1) * pageSize
	if start >= total {
		return []*Species{}, total, pages
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	results := make([]*Species, 0, end-start)
	results = append(results, matches[start:end]...)
	return results, total, pages
}

// AvailableTags returns, for every tag that exists in the catalog, how many
// species match the given selection plus that tag. Tags with count 0 are
// combinations with no data. Query and diameter use the same rules as Filter.
func (c *Catalog) AvailableTags(query string, tags []string, minCM, maxCM int) map[string]int {
	query = strings.ToLower(strings.TrimSpace(query))
	selected := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if tag = strings.TrimSpace(tag); tag != "" {
			selected[tag] = struct{}{}
		}
	}

	knownTags := make(map[string]struct{})
	for _, filterData := range c.filterData {
		for _, tag := range filterData.Tags {
			knownTags[tag] = struct{}{}
		}
	}
	counts := make(map[string]int, len(knownTags))
	for _, species := range c.all {
		filterData := c.filterData[species.Slug]
		if !matchesQueryAndDiameter(species, filterData, query, minCM, maxCM) {
			continue
		}
		present := make(map[string]struct{}, len(filterData.Tags))
		for _, tag := range filterData.Tags {
			present[tag] = struct{}{}
		}
		matchesSelected := true
		for tag := range selected {
			if _, ok := present[tag]; !ok {
				matchesSelected = false
				break
			}
		}
		if !matchesSelected {
			continue
		}
		for tag := range present {
			if _, isSelected := selected[tag]; !isSelected {
				counts[tag]++
			}
		}
	}
	return counts
}

func matchesQueryAndDiameter(species *Species, filterData FilterData, query string, minCM, maxCM int) bool {
	if query != "" && !containsFold(species.NameTrivial, query) && !containsFold(species.NameWissenschaftlich, query) && !containsAnyFold(species.Synonyme, query) {
		return false
	}
	// A zero endpoint means the source has no known diameter. Keep those
	// species visible. Known species remain when their interval overlaps
	// the selected diameter interval.
	return filterData.MinCM == 0 || filterData.MaxCM == 0 || (filterData.MinCM <= maxCM && filterData.MaxCM >= minCM)
}

func (c *Catalog) Letters() []string {
	letters := make([]string, 0, 27)
	for letter := 'A'; letter <= 'Z'; letter++ {
		letters = append(letters, string(letter))
	}
	letters = append(letters, "#")
	return letters
}

func initialLetter(value string) string {
	r, _ := firstRune(nameSortKey(value))
	if r == 0 {
		return ""
	}
	r = unicode.ToUpper(r)
	if r >= 'A' && r <= 'Z' {
		return string(r)
	}
	return "#"
}

func speciesNameLess(left, right *Species) bool {
	leftKey, rightKey := nameSortKey(left.NameTrivial), nameSortKey(right.NameTrivial)
	if leftKey != rightKey {
		return leftKey < rightKey
	}
	leftName, rightName := strings.ToLower(primaryName(left.NameTrivial)), strings.ToLower(primaryName(right.NameTrivial))
	if leftName != rightName {
		return leftName < rightName
	}
	return left.Slug < right.Slug
}

func nameSortKey(value string) string {
	name := primaryName(value)
	name = strings.TrimSpace(name)

	firstNonDigit := 0
	for firstNonDigit < len(name) && name[firstNonDigit] >= '0' && name[firstNonDigit] <= '9' {
		firstNonDigit++
	}
	if firstNonDigit > 0 {
		if number, err := strconv.ParseUint(name[:firstNonDigit], 10, 64); err == nil {
			name = germanNumberWords(number) + name[firstNonDigit:]
		}
	}

	name = strings.ToLower(name)
	return strings.NewReplacer("ä", "a", "ö", "o", "ü", "u", "ß", "ss").Replace(name)
}

func primaryName(value string) string {
	if name, _, found := strings.Cut(value, ","); found {
		return name
	}
	return value
}

func germanNumberWords(number uint64) string {
	if number < 1000 {
		return germanBelowThousand(number)
	}

	var groups []uint64
	for number > 0 {
		groups = append(groups, number%1000)
		number /= 1000
	}
	scales := []string{"", "tausend", "million", "milliarde", "billion", "billiarde", "trillion"}
	parts := make([]string, 0, len(groups))
	for index := len(groups) - 1; index >= 0; index-- {
		group := groups[index]
		if group == 0 {
			continue
		}
		words := germanBelowThousand(group)
		switch index {
		case 0:
			parts = append(parts, words)
		case 1:
			if group == 1 {
				words = "ein"
			}
			parts = append(parts, words+scales[index])
		default:
			if group == 1 {
				words = "eine"
			}
			scale := scales[index]
			if group != 1 {
				scale += "en"
			}
			parts = append(parts, words+" "+scale)
		}
	}
	return strings.Join(parts, " ")
}

func germanBelowThousand(number uint64) string {
	units := []string{"null", "eins", "zwei", "drei", "vier", "fünf", "sechs", "sieben", "acht", "neun", "zehn", "elf", "zwölf", "dreizehn", "vierzehn", "fünfzehn", "sechzehn", "siebzehn", "achtzehn", "neunzehn"}
	tens := []string{"", "", "zwanzig", "dreißig", "vierzig", "fünfzig", "sechzig", "siebzig", "achtzig", "neunzig"}
	if number < 20 {
		return units[number]
	}
	if number < 100 {
		unit, ten := number%10, number/10
		if unit == 0 {
			return tens[ten]
		}
		unitWord := units[unit]
		if unit == 1 {
			unitWord = "ein"
		}
		return unitWord + "und" + tens[ten]
	}
	hundreds, remainder := number/100, number%100
	word := units[hundreds]
	if hundreds == 1 {
		word = "ein"
	}
	word += "hundert"
	if remainder > 0 {
		word += germanBelowThousand(remainder)
	}
	return word
}

func firstRune(value string) (rune, int) {
	for _, r := range value {
		return r, len(string(r))
	}
	return 0, 0
}

func containsFold(value, query string) bool {
	return strings.Contains(strings.ToLower(value), query)
}

func containsAnyFold(values []string, query string) bool {
	for _, value := range values {
		if containsFold(value, query) {
			return true
		}
	}
	return false
}
