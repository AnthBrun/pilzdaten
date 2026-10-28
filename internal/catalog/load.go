package catalog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// Species is one structured entry from species.jsonl. Merkmale is deliberately
// dynamic because fields vary between species and over the source dataset.
type Species struct {
	Slug                 string         `json:"slug"`
	URL                  string         `json:"url"`
	NameTrivial          string         `json:"name_trivial"`
	NameWissenschaftlich string         `json:"name_wissenschaftlich"`
	Synonyme             []string       `json:"synonyme"`
	Merkmale             map[string]any `json:"merkmale"`
	Bilddateien          []string       `json:"bilddateien"`
	Verwechslungen       []string       `json:"verwechslungen"`
	LinksExtern          map[string]any `json:"links_extern"`
	Gattung              string         `json:"-"`
}

type indexEntry struct {
	Slug             string `json:"slug"`
	Name             string `json:"name"`
	Wissenschaftlich string `json:"wissenschaftlich"`
	Gattung          string `json:"gattung"`
}

// Catalog stores species keyed by their stable source slug. The arts index is
// used as the authoritative allow-list so meta/list pages never appear as fungi.
type Catalog struct {
	bySlug map[string]*Species
	count  int
}

func Load(speciesPath, indexPath string) (*Catalog, error) {
	index, err := loadIndex(indexPath)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(speciesPath)
	if err != nil {
		return nil, fmt.Errorf("species-Datei %q öffnen: %w", speciesPath, err)
	}
	defer file.Close()

	catalog := &Catalog{bySlug: make(map[string]*Species, len(index))}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		var species Species
		if err := json.Unmarshal(scanner.Bytes(), &species); err != nil {
			return nil, fmt.Errorf("species-Datei, Zeile %d: %w", line, err)
		}
		entry, isSpecies := index[species.Slug]
		if !isSpecies {
			continue
		}
		if species.NameTrivial == "" {
			species.NameTrivial = entry.Name
		}
		if species.NameWissenschaftlich == "" {
			species.NameWissenschaftlich = entry.Wissenschaftlich
		}
		species.Gattung = entry.Gattung
		catalog.bySlug[species.Slug] = &species
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("species-Datei lesen: %w", err)
	}
	catalog.count = len(catalog.bySlug)
	return catalog, nil
}

func loadIndex(path string) (map[string]indexEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Artenindex %q öffnen: %w", path, err)
	}
	defer file.Close()

	index := make(map[string]indexEntry)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 16*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		var entry indexEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, fmt.Errorf("Artenindex, Zeile %d: %w", line, err)
		}
		if entry.Slug != "" {
			index[entry.Slug] = entry
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Artenindex lesen: %w", err)
	}
	return index, nil
}

func (c *Catalog) Count() int {
	return c.count
}

func (c *Catalog) Get(slug string) (*Species, bool) {
	species, ok := c.bySlug[slug]
	return species, ok
}
