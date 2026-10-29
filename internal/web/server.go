package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"pilzdaten/internal/catalog"
)

//go:embed templates/*.html static/*
var files embed.FS

type Server struct {
	catalog   *catalog.Catalog
	templates *template.Template
}

type pageData struct {
	Count         int
	TotalCount    int
	Letter        string
	Letters       []string
	Page          int
	Pages         int
	Species       []*catalog.Species
	PreviousURL   string
	NextURL       string
	HasPrevious   bool
	HasNext       bool
	Query         string
	Tags          []string
	Selected      map[string]bool
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

type trait struct {
	Name  string
	Value string
}

type image struct {
	URL string
	Alt string
}

type detailData struct {
	DisplayName string
	Aliases     []string
	Species     *catalog.Species
	Traits      []trait
	Images      []image
	Gallery     []image
	Related     []*catalog.Species
}

type searchResult struct {
	Slug                 string `json:"slug"`
	Name                 string `json:"name"`
	ScientificName       string `json:"scientific_name"`
	Edibility            string `json:"edibility"`
	EdibilityLabel       string `json:"edibility_label"`
	EdibilityClass       string `json:"edibility_class"`
	Image                string `json:"image"`
}

type searchResponse struct {
	Count      int            `json:"count"`
	Page       int            `json:"page"`
	TotalPages int            `json:"total_pages"`
	PerPage    int            `json:"per_page"`
	Results    []searchResult `json:"results"`
}

type availableTagsResponse struct {
	Tags []string `json:"tags"`
}

func New(data *catalog.Catalog, imageDir string) *http.ServeMux {
	server := &Server{
		catalog:   data,
		templates: template.Must(template.New("pages").Funcs(template.FuncMap{"imageURL": imageURL, "firstImage": firstImage, "displayName": displayName}).ParseFS(files, "templates/*.html")),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", server.home)
	mux.HandleFunc("GET /art/{slug}", server.detail)
	mux.Handle("GET /api/search", withCacheControl(http.HandlerFunc(server.search), func(string) string { return "no-store" }))
	mux.Handle("GET /api/available-tags", withCacheControl(http.HandlerFunc(server.availableTags), func(string) string { return "no-store" }))
	mux.HandleFunc("GET /api/facets", server.facets)
	mux.Handle("GET /static/", withCacheControl(http.FileServer(http.FS(files)), func(path string) string {
		if strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".js") {
			return "no-cache" // Revalidieren (ETag), damit Updates auf dem Handy ankommen
		}
		return "public, max-age=86400"
	}))
	mux.Handle("GET /media/", http.StripPrefix("/media/", withCacheControl(http.FileServer(http.Dir(imageDir)), func(string) string {
		return "public, max-age=86400"
	})))
	return mux
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	letter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("letter")))
	if len(letter) != 1 || letter[0] < 'A' || letter[0] > 'Z' {
		letter = ""
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	facets := s.catalog.Facets()
	minCM := queryInt(r, "dmin", facets.MinDiameterCM)
	maxCM := queryInt(r, "dmax", facets.MaxDiameterCM)
	if minCM < facets.MinDiameterCM || minCM > facets.MaxDiameterCM {
		minCM = facets.MinDiameterCM
	}
	if maxCM < facets.MinDiameterCM || maxCM > facets.MaxDiameterCM {
		maxCM = facets.MaxDiameterCM
	}
	if minCM > maxCM {
		minCM, maxCM = maxCM, minCM
	}
	tags := splitTags(r.URL.Query().Get("f"))
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	species, count, pages := s.catalog.Filter(query, tags, minCM, maxCM, letter, page, 50)
	selected := make(map[string]bool, len(tags))
	for _, tag := range tags {
		selected[tag] = true
	}
	activeFilters := make([]filterChip, 0, len(tags))
	labels := make(map[string]string)
	for _, group := range facets.Groups {
		for _, option := range group.Options {
			labels[option.Tag] = option.Label
		}
	}
	for _, tag := range tags {
		chipLabel := labels[tag]
		if chipLabel == "" {
			chipLabel = label(tag)
		}
		activeFilters = append(activeFilters, filterChip{Tag: tag, Label: chipLabel})
	}
	values := r.URL.Query()
	values.Del("page")
	baseQuery := values.Encode()
	previousURL, nextURL := "?", "?"
	if baseQuery != "" {
		previousURL, nextURL = "?"+baseQuery+"&", "?"+baseQuery+"&"
	}
	data := pageData{
		Count: count, TotalCount: s.catalog.Count(), Letter: letter, Letters: s.catalog.Letters(),
		Page: page, Pages: pages, Species: species,
		PreviousURL: previousURL + "page=" + strconv.Itoa(page-1), NextURL: nextURL + "page=" + strconv.Itoa(page+1),
		HasPrevious: page > 1, HasNext: page < pages,
		Query: query, Tags: tags, Selected: selected, MinCM: minCM, MaxCM: maxCM, Facets: facets,
		ActiveFilters: activeFilters, HasFilters: len(tags) > 0 || query != "" || minCM != facets.MinDiameterCM || maxCM != facets.MaxDiameterCM,
		SuggestMore: len(tags) > 0 && count > 50,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Seite konnte nicht gerendert werden", http.StatusInternalServerError)
	}
}

func queryInt(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	return value
}

func withCacheControl(next http.Handler, policy func(path string) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", policy(r.URL.Path))
		next.ServeHTTP(w, r)
	})
}

// displayName gibt den ersten Trivialnamen eines Komma-getrennten Namensstrings zurück.
func displayName(nameTrivial string) string {
	trimmed := strings.TrimSpace(nameTrivial)
	if i := strings.Index(trimmed, ","); i >= 0 {
		return strings.TrimSpace(trimmed[:i])
	}
	return trimmed
}

func splitTags(value string) []string {
	seen := make(map[string]struct{})
	tags := make([]string, 0)
	for _, tag := range strings.Split(value, ",") {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; !ok {
			seen[tag] = struct{}{}
			tags = append(tags, tag)
		}
	}
	return tags
}

func (s *Server) facets(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(s.catalog.Facets()); err != nil {
		http.Error(w, "Filterdefinition konnte nicht ausgegeben werden", http.StatusInternalServerError)
	}
}

func (s *Server) detail(w http.ResponseWriter, r *http.Request) {
	species, ok := s.catalog.Get(r.PathValue("slug"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	data := detailData{Species: species, DisplayName: displayName(species.NameTrivial)}
	for _, alias := range strings.Split(species.NameTrivial, ",") {
		alias = strings.TrimSpace(alias)
		if alias != "" && alias != data.DisplayName {
			data.Aliases = append(data.Aliases, alias)
		}
	}
	data.Aliases = append(data.Aliases, species.Synonyme...)
	for key, value := range species.Merkmale {
		if rendered := displayValue(value); rendered != "" {
			data.Traits = append(data.Traits, trait{Name: label(key), Value: rendered})
		}
	}
	sort.Slice(data.Traits, func(i, j int) bool {
		return strings.ToLower(data.Traits[i].Name) < strings.ToLower(data.Traits[j].Name)
	})
	seenImages := make(map[string]bool)
	addPhoto := func(path string) {
		if path == "" || !strings.HasSuffix(strings.ToLower(path), ".jpg") || seenImages[path] {
			return
		}
		seenImages[path] = true
		data.Images = append(data.Images, image{URL: imageURL(path), Alt: species.NameTrivial})
	}
	addPhoto(species.BestImage)
	for _, imagePath := range species.Bilddateien {
		addPhoto(imagePath)
	}
	if len(data.Images) > 1 {
		data.Gallery = data.Images[1:]
	}
	for _, slug := range species.Verwechslungen {
		if related, found := s.catalog.Get(slug); found {
			data.Related = append(data.Related, related)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "detail.html", data); err != nil {
		http.Error(w, "Artseite konnte nicht gerendert werden", http.StatusInternalServerError)
	}
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	minCM, maxCM := diameterRange(r, s.catalog.Facets())
	page := queryInt(r, "page", 1)
	if page < 1 {
		page = 1
	}
	const perPage = 50
	letter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("letter")))
	if len(letter) != 1 || letter[0] < 'A' || letter[0] > 'Z' {
		letter = ""
	}
	results, count, pages := s.catalog.Filter(r.URL.Query().Get("q"), splitTags(r.URL.Query().Get("f")), minCM, maxCM, letter, page, perPage)
	response := searchResponse{Count: count, Page: page, TotalPages: pages, PerPage: perPage, Results: make([]searchResult, 0, len(results))}
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
			Edibility: species.Edibility, EdibilityLabel: species.EdibilityLabel, EdibilityClass: species.EdibilityClass,
			Image: imageURLValue,
		})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Suchergebnis konnte nicht ausgegeben werden", http.StatusInternalServerError)
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(availableTagsResponse{Tags: tags}); err != nil {
		http.Error(w, "Verfügbare Merkmale konnten nicht ausgegeben werden", http.StatusInternalServerError)
	}
}

func diameterRange(r *http.Request, facets catalog.Facets) (int, int) {
	minCM := boundedQueryInt(r, "dmin", facets.MinDiameterCM, facets.MinDiameterCM, facets.MaxDiameterCM)
	maxCM := boundedQueryInt(r, "dmax", facets.MaxDiameterCM, facets.MinDiameterCM, facets.MaxDiameterCM)
	if minCM > maxCM {
		minCM, maxCM = maxCM, minCM
	}
	return minCM, maxCM
}

func boundedQueryInt(r *http.Request, key string, fallback, min, max int) int {
	value := queryInt(r, key, fallback)
	if value < min || value > max {
		return fallback
	}
	return value
}

func displayValue(value any) string {
	switch value := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	case []any:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			if part := displayValue(item); part != "" {
				parts = append(parts, part)
			}
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprint(value)
		}
		return string(encoded)
	default:
		return fmt.Sprint(value)
	}
}

func label(value string) string {
	value = strings.ReplaceAll(value, "_", " ")
	runes := []rune(value)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return string(runes)
}

func imageURL(path string) string {
	parts := strings.Split(strings.TrimLeft(path, "/"), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return "/media/" + strings.Join(parts, "/")
}

// firstImage returns the first real photo (a .jpg) of a species; the source
// dataset starts each image list with small 15x15 gif markers. If the species
// has no .jpg, the first entry is used as a fallback.
func firstImage(paths []string) string {
	for _, path := range paths {
		if strings.HasSuffix(strings.ToLower(path), ".jpg") {
			return path
		}
	}
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}
