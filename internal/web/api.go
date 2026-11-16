package web

import (
	"encoding/json"
	"net/http"
	"sort"

	"pilzdaten/internal/catalog"
	"pilzdaten/internal/i18n"
)

// GET /api/search – Treffer einer Seite, Antwortform siehe searchResponse.
type searchResponse struct {
	Count      int            `json:"count"`
	Page       int            `json:"page"`
	TotalPages int            `json:"total_pages"`
	PerPage    int            `json:"per_page"`
	Results    []searchResult `json:"results"`
}

type searchResult struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	ScientificName string `json:"scientific_name"`
	Edibility      string `json:"edibility"`
	EdibilityLabel string `json:"edibility_label"`
	EdibilityClass string `json:"edibility_class"`
	Image          string `json:"image"`
}

// GET /api/available-tags – Tags, die zusammen mit der aktuellen Auswahl
// noch mindestens eine Art liefern. Das Frontend graut die übrigen aus.
type availableTagsResponse struct {
	Tags []string `json:"tags"`
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	lang := s.requestLang(w, r)
	minCM, maxCM := diameterRange(r, s.catalog.Facets())
	page := queryPage(r)
	letter := validLetter(r.URL.Query().Get("letter"))
	results, count, pages := s.catalog.Filter(r.URL.Query().Get("q"), splitTags(r.URL.Query().Get("f")), minCM, maxCM, letter, page, pageSize)
	response := searchResponse{Count: count, Page: page, TotalPages: pages, PerPage: pageSize, Results: make([]searchResult, 0, len(results))}
	for _, species := range results {
		response.Results = append(response.Results, s.cardResult(lang, species))
	}
	w.Header().Add("Vary", "Cookie, Accept-Language")
	writeJSON(w, response)
}

// cardResult baut die Kartendaten einer Art (Suche, Foto-Vorschläge).
func (s *Server) cardResult(lang string, species *catalog.Species) searchResult {
	imagePath := species.BestImage
	if imagePath == "" {
		imagePath = firstImage(species.Bilddateien)
	}
	imageURLValue := ""
	if imagePath != "" {
		imageURLValue = imageURL(imagePath)
	}
	return searchResult{
		Slug: species.Slug, Name: displayName(s.t(lang, species.NameTrivial)),
		ScientificName: species.NameWissenschaftlich,
		Edibility:      species.Edibility, EdibilityLabel: s.t(lang, species.EdibilityLabel), EdibilityClass: species.EdibilityClass,
		Image: imageURLValue,
	}
}

func (s *Server) availableTags(w http.ResponseWriter, r *http.Request) {
	minCM, maxCM := diameterRange(r, s.catalog.Facets())
	counts := s.catalog.AvailableTags(r.URL.Query().Get("q"), splitTags(r.URL.Query().Get("f")), minCM, maxCM)
	tags := make([]string, 0, len(counts))
	for tag, count := range counts {
		if count > 0 {
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)
	writeJSON(w, availableTagsResponse{Tags: tags})
}

// GET /api/facets – alle Filtergruppen mit Labels und Anzahlen.
func (s *Server) facets(w http.ResponseWriter, r *http.Request) {
	lang := s.requestLang(w, r)
	w.Header().Add("Vary", "Cookie, Accept-Language")
	writeJSON(w, s.localFacets(lang))
}

// localFacets liefert die Filtergruppen mit übersetzten Labels und Titeln.
func (s *Server) localFacets(lang string) catalog.Facets {
	facets := s.catalog.Facets()
	if lang == i18n.Source {
		return facets
	}
	groups := make([]catalog.FacetGroup, len(facets.Groups))
	for groupIndex, group := range facets.Groups {
		group.Title = s.t(lang, group.Title)
		options := make([]catalog.FacetOption, len(group.Options))
		for optionIndex, option := range group.Options {
			option.Label, option.Title = s.t(lang, option.Label), s.t(lang, option.Title)
			options[optionIndex] = option
		}
		group.Options = options
		groups[groupIndex] = group
	}
	facets.Groups = groups
	return facets
}

var msgWriteFailed = "Antwort konnte nicht ausgegeben werden"

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, msgWriteFailed, http.StatusInternalServerError)
	}
}
