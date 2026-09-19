package server

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/mlkad/groupie-tracker/internal/api"
)

type Server struct {
	cache     *api.Cache
	templates map[string]*template.Template
}

func New(cache *api.Cache) (*Server, error) {
	pages := []string{"home.html", "artist.html", "search.html", "error.html"}
	templates := make(map[string]*template.Template)

	for _, page := range pages {
		t, err := template.New("base.html").Funcs(template.FuncMap{"formatLocation": formatLocation}).ParseFiles("web/templates/base.html", "web/templates/"+page)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", page, err)
		}
		templates[page] = t
	}
	return &Server{
		cache:     cache,
		templates: templates,
	}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("GET /artist/{id}", s.artist)
	mux.HandleFunc("GET /search", s.search)
	mux.HandleFunc("GET /suggest", s.suggest)
	mux.HandleFunc("GET /filter", s.filter)

	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	return recoverPanic(mux)
}

func (s *Server) render(w http.ResponseWriter, status int, page string, data any) {
	t, ok := s.templates[page]
	if !ok {
		log.Printf("template %s not found", page)
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	buf := new(bytes.Buffer)
	err := t.ExecuteTemplate(buf, "base", data)
	if err != nil {
		log.Printf("rendering %s: %v", page, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w)

}
