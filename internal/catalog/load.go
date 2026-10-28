package catalog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
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

type FilterData struct {
	Tags  []string `json:"tags"`
	MinCM int      `json:"hut_durchmesser_min_cm"`
	MaxCM int      `json:"hut_durchmesser_max_cm"`
}

type Facets struct {
	MaxDiameterCM int          `json:"max_diameter_cm"`
	MinDiameterCM int          `json:"min_diameter_cm"`
	Groups        []FacetGroup `json:"groups"`
}

type FacetGroup struct {
	Group   string        `json:"group"`
	Title   string        `json:"title"`
	Options []FacetOption `json:"options"`
}

type FacetOption struct {
	Tag   string `json:"tag"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// Catalog stores species keyed by their stable source slug. The arts index is
// used as the authoritative allow-list so meta/list pages never appear as fungi.
type Catalog struct {
	bySlug     map[string]*Species
	all        []*Species
	filterData map[string]FilterData
	facets     Facets
	count      int
}

func Load(speciesPath, indexPath string) (*Catalog, error) {
	return LoadWithFilters(speciesPath, indexPath, "", "")
}

func LoadWithFilters(speciesPath, indexPath, filterTagsPath, facetsPath string) (*Catalog, error) {
	index, err := loadIndex(indexPath)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(speciesPath)
	if err != nil {
		return nil, fmt.Errorf("species-Datei %q öffnen: %w", speciesPath, err)
	}
	defer file.Close()

	catalog := &Catalog{bySlug: make(map[string]*Species, len(index)), filterData: make(map[string]FilterData), facets: Facets{MinDiameterCM: 1, MaxDiameterCM: 25}}
	if filterTagsPath != "" {
		catalog.filterData, err = loadFilterTags(filterTagsPath)
		if err != nil {
			return nil, err
		}
	}
	if facetsPath != "" {
		if err := loadFacets(facetsPath, &catalog.facets); err != nil {
			return nil, err
		}
	}
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
		catalog.all = append(catalog.all, &species)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("species-Datei lesen: %w", err)
	}
	catalog.count = len(catalog.bySlug)
	sort.Slice(catalog.all, func(i, j int) bool {
		left := strings.ToLower(catalog.all[i].NameTrivial)
		right := strings.ToLower(catalog.all[j].NameTrivial)
		if left == right {
			return catalog.all[i].Slug < catalog.all[j].Slug
		}
		return left < right
	})
	return catalog, nil
}

func loadFilterTags(path string) (map[string]FilterData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Filtermerkmale %q öffnen: %w", path, err)
	}
	defer file.Close()

	filters := make(map[string]FilterData)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 16*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		var entry struct {
			Slug string `json:"slug"`
			FilterData
		}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, fmt.Errorf("Filtermerkmale, Zeile %d: %w", line, err)
		}
		if entry.Slug != "" {
			current := filters[entry.Slug]
			knownTags := make(map[string]struct{}, len(current.Tags)+len(entry.Tags))
			for _, tag := range current.Tags {
				knownTags[tag] = struct{}{}
			}
			for _, tag := range entry.Tags {
				if _, exists := knownTags[tag]; !exists {
					current.Tags = append(current.Tags, tag)
					knownTags[tag] = struct{}{}
				}
			}
			if entry.MinCM != 0 && (current.MinCM == 0 || entry.MinCM < current.MinCM) {
				current.MinCM = entry.MinCM
			}
			if entry.MaxCM != 0 && (current.MaxCM == 0 || entry.MaxCM > current.MaxCM) {
				current.MaxCM = entry.MaxCM
			}
			filters[entry.Slug] = current
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Filtermerkmale lesen: %w", err)
	}
	return filters, nil
}

func loadFacets(path string, facets *Facets) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("Filterdefinition %q öffnen: %w", path, err)
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(facets); err != nil {
		return fmt.Errorf("Filterdefinition %q lesen: %w", path, err)
	}
	return nil
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

func (c *Catalog) Facets() Facets {
	return c.facets
}
