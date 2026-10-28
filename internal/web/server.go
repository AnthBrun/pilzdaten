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
	Count       int
	TotalCount  int
	Letter      string
	Letters     []string
	Page        int
	Pages       int
	Species     []*catalog.Species
	PreviousURL string
	NextURL     string
	HasPrevious bool
	HasNext     bool
	Query       string
	Tags        []string
	Selected    map[string]bool
	MinCM       int
	MaxCM       int
	Facets      catalog.Facets
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
	Species *catalog.Species
	Traits  []trait
	Images  []image
	Related []*catalog.Species
}

type searchResult struct {
	Slug                 string `json:"slug"`
	NameTrivial          string `json:"name_trivial"`
	NameWissenschaftlich string `json:"name_wissenschaftlich"`
}

func New(data *catalog.Catalog, imageDir string) *http.ServeMux {
	server := &Server{
		catalog:   data,
		templates: template.Must(template.New("pages").Funcs(template.FuncMap{"imageURL": imageURL}).ParseFS(files, "templates/*.html")),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", server.home)
	mux.HandleFunc("GET /art/{slug}", server.detail)
	mux.HandleFunc("GET /api/search", server.search)
	mux.HandleFunc("GET /api/facets", server.facets)
	mux.Handle("GET /static/", http.FileServer(http.FS(files)))
	mux.Handle("GET /media/", http.StripPrefix("/media/", http.FileServer(http.Dir(imageDir))))
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
	data := detailData{Species: species}
	for key, value := range species.Merkmale {
		if rendered := displayValue(value); rendered != "" {
			data.Traits = append(data.Traits, trait{Name: label(key), Value: rendered})
		}
	}
	sort.Slice(data.Traits, func(i, j int) bool {
		return strings.ToLower(data.Traits[i].Name) < strings.ToLower(data.Traits[j].Name)
	})
	for _, imagePath := range species.Bilddateien {
		if imagePath != "" {
			data.Images = append(data.Images, image{URL: imageURL(imagePath), Alt: species.NameTrivial})
		}
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
	results := s.catalog.Search(r.URL.Query().Get("q"), 30)
	response := make([]searchResult, 0, len(results))
	for _, species := range results {
		response = append(response, searchResult{
			Slug: species.Slug, NameTrivial: species.NameTrivial,
			NameWissenschaftlich: species.NameWissenschaftlich,
		})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Suchergebnis konnte nicht ausgegeben werden", http.StatusInternalServerError)
	}
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
