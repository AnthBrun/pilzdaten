package catalog

import (
	"sort"
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

	type match struct {
		species *Species
		score   int
	}
	matches := make([]match, 0)
	for _, species := range c.all {
		if letter != "" && initialLetter(species.NameTrivial) != letter {
			continue
		}
		if query != "" && !containsFold(species.NameTrivial, query) && !containsFold(species.NameWissenschaftlich, query) && !containsAnyFold(species.Synonyme, query) {
			continue
		}
		filterData := c.filterData[species.Slug]
		present := make(map[string]struct{}, len(filterData.Tags))
		for _, tag := range filterData.Tags {
			present[tag] = struct{}{}
		}
		score := 0
		matchesTags := true
		for tag := range selected {
			if _, ok := present[tag]; !ok {
				matchesTags = false
				break
			}
			score++
		}
		if !matchesTags {
			continue
		}
		// A zero endpoint means the source has no known diameter. Keep those
		// species visible. Known species remain when their interval overlaps
		// the selected diameter interval.
		if filterData.MinCM != 0 && filterData.MaxCM != 0 && (filterData.MinCM > maxCM || filterData.MaxCM < minCM) {
			continue
		}
		matches = append(matches, match{species: species, score: score})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		left, right := strings.ToLower(matches[i].species.NameTrivial), strings.ToLower(matches[j].species.NameTrivial)
		if left == right {
			return matches[i].species.Slug < matches[j].species.Slug
		}
		return left < right
	})

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
	for _, item := range matches[start:end] {
		results = append(results, item.species)
	}
	return results, total, pages
}

func (c *Catalog) Letters() []string {
	letters := make([]string, 0, 26)
	for letter := 'A'; letter <= 'Z'; letter++ {
		letters = append(letters, string(letter))
	}
	return letters
}

func initialLetter(value string) string {
	r, _ := firstRune(value)
	if r == 0 {
		return ""
	}
	return string(unicode.ToUpper(r))
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
