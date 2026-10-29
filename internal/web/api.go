package web

import (
	"encoding/json"
	"net/http"
	"sort"
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
	minCM, maxCM := diameterRange(r, s.catalog.Facets())
	page := queryPage(r)
	letter := validLetter(r.URL.Query().Get("letter"))
	results, count, pages := s.catalog.Filter(r.URL.Query().Get("q"), splitTags(r.URL.Query().Get("f")), minCM, maxCM, letter, page, pageSize)
	response := searchResponse{Count: count, Page: page, TotalPages: pages, PerPage: pageSize, Results: make([]searchResult, 0, len(results))}
	for _, species := range results {
		imagePath := species.BestImage
		if imagePath == "" {
			imagePath = firstImage(species.Bilddateien)
		}
		imageURLValue := ""
		if imagePath != "" {
			imageURLValue = imageURL(imagePath)
		}
		response.Results = append(response.Results, searchResult{
			Slug: species.Slug, Name: displayName(species.NameTrivial),
			ScientificName: species.NameWissenschaftlich,
			Edibility:      species.Edibility, EdibilityLabel: species.EdibilityLabel, EdibilityClass: species.EdibilityClass,
			Image: imageURLValue,
		})
	}
	writeJSON(w, response)
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
func (s *Server) facets(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.catalog.Facets())
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, "Antwort konnte nicht ausgegeben werden", http.StatusInternalServerError)
	}
}
