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
	Letter      string
	Letters     []string
	Page        int
	Pages       int
	Species     []*catalog.Species
	Previous    int
	Next        int
	HasPrevious bool
	HasNext     bool
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
		templates: template.Must(template.ParseFS(files, "templates/*.html")),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", server.home)
	mux.HandleFunc("GET /art/{slug}", server.detail)
	mux.HandleFunc("GET /api/search", server.search)
	mux.Handle("GET /static/", http.FileServer(http.FS(files)))
	mux.Handle("GET /media/", http.StripPrefix("/media/", http.FileServer(http.Dir(imageDir))))
	return mux
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	letter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("letter")))
	if len(letter) != 1 || letter[0] < 'A' || letter[0] > 'Z' {
		letter = "A"
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	species, pages := s.catalog.List(letter, page, 50)
	data := pageData{
		Count: s.catalog.Count(), Letter: letter, Letters: s.catalog.Letters(),
		Page: page, Pages: pages, Species: species,
		Previous: page - 1, Next: page + 1, HasPrevious: page > 1, HasNext: page < pages,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Seite konnte nicht gerendert werden", http.StatusInternalServerError)
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
