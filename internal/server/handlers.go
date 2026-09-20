package server

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"sort"
	"strconv"

	"github.com/mlkad/groupie-tracker/internal/api"
)

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	data := homeData{
		Artists:   s.cache.Artists(),
		Query:     "",
		Locations: s.cache.AllLocations(),
		Selected:  map[string]bool{},
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

func (s *Server) filter(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	creationFrom := q.Get("creation_from")
	creationTo := q.Get("creation_to")
	albumFrom := q.Get("album_from")
	albumTo := q.Get("album_to")
	membersFrom := q.Get("members_from")
	membersTo := q.Get("members_to")
	membersExact := q.Get("members_exact")
	location := q["location"]
	sortBy := q.Get("sort")

	f := api.ArtistFilter{
		CreationFrom: atoiFn(creationFrom, 0),
		CreationTo:   atoiFn(creationTo, math.MaxInt),
		AlbumFrom:    atoiFn(albumFrom, 0),
		AlbumTo:      atoiFn(albumTo, math.MaxInt),
		MembersFrom:  atoiFn(membersFrom, 0),
		MembersTo:    atoiFn(membersTo, math.MaxInt),
		MembersExact: atoiFn(membersExact, 0),
		Locations:    location,
	}

	artists := s.cache.Filter(f)

	switch sortBy {
	case "name-asc":
		sort.Slice(artists, func(i, j int) bool {
			return artists[i].Name < artists[j].Name
		})
	case "name-desc":
		sort.Slice(artists, func(i, j int) bool {
			return artists[i].Name > artists[j].Name
		})
	case "creation-asc":
		sort.Slice(artists, func(i, j int) bool {
			return artists[i].CreationDate < artists[j].CreationDate
		})
	case "creation-desc":
		sort.Slice(artists, func(i, j int) bool {
			return artists[i].CreationDate > artists[j].CreationDate
		})
	case "members-asc":
		sort.Slice(artists, func(i, j int) bool {
			return len(artists[i].Members) < len(artists[j].Members)
		})
	case "members-desc":
		sort.Slice(artists, func(i, j int) bool {
			return len(artists[i].Members) > len(artists[j].Members)
		})
	default:
		sort.Slice(artists, func(i, j int) bool {
			return artists[i].Name < artists[j].Name
		})
	}

	const perPage = 8

	page := atoiFn(q.Get("page"), 1)

	artistsPage, totalPages := paginate(artists, page, perPage)

	data := homeData{
		Artists:      artistsPage,
		Page:         page,
		TotalPages:   totalPages,
		CreationFrom: creationFrom,
		CreationTo:   creationTo,
		AlbumFrom:    albumFrom,
		AlbumTo:      albumTo,
		MembersFrom:  membersFrom,
		MembersTo:    membersTo,
		MembersExact: membersExact,
		Sort:         sortBy,
		Locations:    s.cache.AllLocations(),
		Selected:     selectedSet(location),
		Query:        "",
	}

	s.render(w, http.StatusOK, "home.html", data)
}
