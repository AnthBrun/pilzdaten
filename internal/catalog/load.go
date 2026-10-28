package catalog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
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
	BestImage            string         `json:"-"`
	Edibility            string         `json:"-"`
	EdibilityLabel       string         `json:"-"`
	EdibilityClass       string         `json:"-"`
	EdibilityWarning     bool           `json:"-"`
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
	Tag         string `json:"tag"`
	Label       string `json:"label"`
	Count       int    `json:"count"`
	Title       string `json:"title,omitempty"`
	Image       string `json:"image,omitempty"`
	ActiveImage string `json:"active_image,omitempty"`
}

type Pictogram struct {
	Title       string `json:"title"`
	Image       string `json:"img"`
	ActiveImage string `json:"img_active"`
}

// Catalog stores species keyed by their stable source slug. The arts index is
// used as the authoritative allow-list so meta/list pages never appear as fungi.
type Catalog struct {
	bySlug     map[string]*Species
	all        []*Species
	filterData map[string]FilterData
	bestImages map[string]string
	facets     Facets
	count      int
}

func Load(speciesPath, indexPath string) (*Catalog, error) {
	return LoadWithFilters(speciesPath, indexPath, "", "")
}

func LoadWithFilters(speciesPath, indexPath, filterTagsPath, facetsPath string) (*Catalog, error) {
	return LoadWithAssets(speciesPath, indexPath, filterTagsPath, facetsPath, "", "")
}

// LoadWithAssets additionally loads preferred card/detail photos and the
// pictogram metadata used to render the visual facet controls.
func LoadWithAssets(speciesPath, indexPath, filterTagsPath, facetsPath, bestImagesPath, pictogramMapPath string) (*Catalog, error) {
	index, err := loadIndex(indexPath)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(speciesPath)
	if err != nil {
		return nil, fmt.Errorf("species-Datei %q öffnen: %w", speciesPath, err)
	}
	defer file.Close()

	catalog := &Catalog{bySlug: make(map[string]*Species, len(index)), filterData: make(map[string]FilterData), bestImages: make(map[string]string), facets: Facets{MinDiameterCM: 1, MaxDiameterCM: 25}}
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
	if bestImagesPath != "" {
		catalog.bestImages, err = loadBestImages(bestImagesPath)
		if err != nil {
			return nil, err
		}
	}
	if pictogramMapPath != "" {
		pictograms, err := loadPictograms(pictogramMapPath)
		if err != nil {
			return nil, err
		}
		catalog.applyPictograms(pictograms)
	}
	catalog.promoteEdibilityFacet()
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
		species.BestImage = catalog.bestImages[species.Slug]
		applyEdibility(&species, catalog.filterData[species.Slug].Tags)
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

func loadBestImages(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("beste Bilder %q öffnen: %w", path, err)
	}
	defer file.Close()
	images := make(map[string]string)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 16*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		var entry struct {
			Slug  string `json:"slug"`
			Image string `json:"image"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, fmt.Errorf("beste Bilder, Zeile %d: %w", line, err)
		}
		if entry.Slug != "" && entry.Image != "" {
			images[entry.Slug] = entry.Image
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("beste Bilder lesen: %w", err)
	}
	return images, nil
}

func loadPictograms(path string) (map[string]Pictogram, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Piktogramm-Zuordnung %q öffnen: %w", path, err)
	}
	defer file.Close()
	pictograms := make(map[string]Pictogram)
	if err := json.NewDecoder(file).Decode(&pictograms); err != nil {
		return nil, fmt.Errorf("Piktogramm-Zuordnung %q lesen: %w", path, err)
	}
	return pictograms, nil
}

func (c *Catalog) applyPictograms(pictograms map[string]Pictogram) {
	for groupIndex := range c.facets.Groups {
		for optionIndex := range c.facets.Groups[groupIndex].Options {
			option := &c.facets.Groups[groupIndex].Options[optionIndex]
			if pictogram, ok := pictograms[option.Tag]; ok {
				option.Title = html.UnescapeString(pictogram.Title)
				option.Image = pictogram.Image
				option.ActiveImage = pictogram.ActiveImage
			}
		}
	}
}

func (c *Catalog) promoteEdibilityFacet() {
	for index, group := range c.facets.Groups {
		if group.Group == "essbarkeit" {
			if index > 0 {
				c.facets.Groups = append([]FacetGroup{group}, append(c.facets.Groups[:index], c.facets.Groups[index+1:]...)...)
			}
			return
		}
	}
}

func applyEdibility(species *Species, tags []string) {
	known := make(map[string]bool, len(tags))
	for _, tag := range tags {
		known[tag] = true
	}
	switch {
	case known["tot"]:
		species.Edibility, species.EdibilityLabel, species.EdibilityClass, species.EdibilityWarning = "tot", "Tödlich giftig", "deadly", true
	case known["giftig"]:
		species.Edibility, species.EdibilityLabel, species.EdibilityClass, species.EdibilityWarning = "giftig", "Giftig", "poisonous", true
	case known["schlecht"]:
		species.Edibility, species.EdibilityLabel, species.EdibilityClass = "schlecht", "Ungenießbar / schlecht", "poor"
	case known["hervorragend"]:
		species.Edibility, species.EdibilityLabel, species.EdibilityClass = "hervorragend", "Hervorragend essbar", "excellent"
	case known["gut"]:
		species.Edibility, species.EdibilityLabel, species.EdibilityClass = "gut", "Gut essbar", "edible"
	default:
		species.Edibility, species.EdibilityLabel, species.EdibilityClass = "", "Essbarkeit nicht erfasst", "unknown"
	}
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
