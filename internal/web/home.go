package web

import (
	"net/http"
	"strconv"
	"strings"

	"pilzdaten/internal/catalog"
)

// pageData füllt templates/index.html. Die Trefferliste wird serverseitig
// vorgerendert (funktioniert ohne JS) und danach von js/main.js aktualisiert.
type pageData struct {
	Count         int
	Page          int
	Pages         int
	Species       []*catalog.Species
	PreviousURL   string
	NextURL       string
	HasPrevious   bool
	HasNext       bool
	Query         string
	MinCM         int
	MaxCM         int
	Facets        catalog.Facets
	ActiveFilters []filterChip
	HasFilters    bool
	SuggestMore   bool
}

type filterChip struct {
	Tag   string
	Label string
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	facets := s.catalog.Facets()
	page := queryPage(r)
	minCM, maxCM := diameterRange(r, facets)
	tags := splitTags(r.URL.Query().Get("f"))
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	letter := validLetter(r.URL.Query().Get("letter"))
	species, count, pages := s.catalog.Filter(query, tags, minCM, maxCM, letter, page, pageSize)

	labels := make(map[string]string)
	for _, group := range facets.Groups {
		for _, option := range group.Options {
			labels[option.Tag] = option.Label
		}
	}
	activeFilters := make([]filterChip, 0, len(tags))
	for _, tag := range tags {
		chipLabel := labels[tag]
		if chipLabel == "" {
			chipLabel = label(tag)
		}
		activeFilters = append(activeFilters, filterChip{Tag: tag, Label: chipLabel})
	}

	values := r.URL.Query()
	values.Del("page")
	pageURL := func(target int) string {
		values.Set("page", strconv.Itoa(target))
		return "?" + values.Encode()
	}
	s.render(w, "index.html", pageData{
		Count: count, Page: page, Pages: pages, Species: species,
		PreviousURL: pageURL(page - 1), NextURL: pageURL(page + 1),
		HasPrevious: page > 1, HasNext: page < pages,
		Query: query, MinCM: minCM, MaxCM: maxCM, Facets: facets,
		ActiveFilters: activeFilters,
		HasFilters:    len(tags) > 0 || query != "" || minCM != facets.MinDiameterCM || maxCM != facets.MaxDiameterCM,
		SuggestMore:   len(tags) > 0 && count > pageSize,
	})
}
