package web

import (
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"

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

type searchResult struct {
	Slug                 string `json:"slug"`
	NameTrivial          string `json:"name_trivial"`
	NameWissenschaftlich string `json:"name_wissenschaftlich"`
}

func New(data *catalog.Catalog) *http.ServeMux {
	server := &Server{
		catalog:   data,
		templates: template.Must(template.ParseFS(files, "templates/*.html")),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", server.home)
	mux.HandleFunc("GET /api/search", server.search)
	mux.Handle("GET /static/", http.FileServer(http.FS(files)))
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
