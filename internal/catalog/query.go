package catalog

import (
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
