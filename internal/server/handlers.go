package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"

	"github.com/mlkad/groupie-tracker/internal/api"
)

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	data := homeData{
		Artists: s.cache.Artists(),
		Query:   "",
	}
	s.render(w, http.StatusOK, "home.html", data)
}

func (s *Server) artist(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		s.clientError(w, http.StatusBadRequest)
		return
	}

	a, err := s.cache.Artist(id)
	if err != nil {
		if errors.Is(err, api.ErrNotFound) {
			s.clientError(w, http.StatusNotFound)
			return
		}
		s.serverError(w, err)
		return
	}

	rel, err := s.cache.Relation(id)
	if err != nil && !errors.Is(err, api.ErrNotFound) {
		s.serverError(w, err)
		return
	}

	concerts := make([]concert, 0, len(rel.DatesLocations))

	for loc, dates := range rel.DatesLocations {
		cleaned := make([]string, 0, len(dates))
		for _, d := range dates {
			cleaned = append(cleaned, cleanDate(d))
		}
		concerts = append(concerts, concert{
			Location: formatLocation(loc),
			Dates:    cleaned,
		})
	}

	sort.Slice(concerts, func(i, j int) bool {
		return concerts[i].Location < concerts[j].Location
	})

	data := artistData{
		Artist:   a,
		Concerts: concerts,
		Query:    "",
	}
	s.render(w, http.StatusOK, "artist.html", data)

}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	data := searchData{
		Artists: s.cache.Search(q),
		Query:   q,
	}
	s.render(w, http.StatusOK, "search.html", data)
}

func (s *Server) suggest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	found := s.cache.Search(q)

	out := make([]suggestion, 0, len(found)) //не через var потому что null срез. null ломает js: list.forEach
	for i, a := range found {
		if i >= 8 {
			break
		}
		out = append(out, suggestion{
			ID:   a.ID,
			Name: a.Name,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		s.serverError(w, err)
	}
}
