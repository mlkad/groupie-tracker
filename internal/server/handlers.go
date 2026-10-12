package server

import (
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"

	"github.com/mlkad/groupie-tracker/internal/api"
	"github.com/mlkad/groupie-tracker/internal/models"
)

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

		name := formatLocation(loc) //напр "mainz-germany" - "Mainz, Germany"
		c := concert{
			Location: name,
			Dates:    cleaned,
		}

		coords, err := s.geo.Geocode(r.Context(), name) // спросить lon, lat
		if err != nil {
			log.Printf("geocoding %q: %v", name, err)
		} else {
			c.Lat = coords.Lat
			c.Lon = coords.Lon
			c.Found = true
		}

		concerts = append(concerts, c)
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
	results := s.cache.Search(q)

	artists := make([]models.Artist, 0, len(results))
	for _, result := range results {
		artists = append(artists, result.Artist)
	}

	data := searchData{
		Artists: artists,
		Query:   q,
	}
	s.render(w, http.StatusOK, "search.html", data)
}

func (s *Server) suggest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	found := s.cache.Search(q)

	out := make([]suggestion, 0, 8)
	for _, result := range found {
		for _, matchType := range result.Types {
			if len(out) == 8 {
				break
			}

			out = append(out, suggestion{
				ID:   result.Artist.ID,
				Name: result.Artist.Name,
				Type: matchType,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		s.serverError(w, err)
	}
}

func (s *Server) artists(w http.ResponseWriter, r *http.Request) {
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

	totalArtists := len(artists)
	const perPage = 8

	page := atoiFn(q.Get("page"), 1)

	artistsPage, totalPages := paginate(artists, page, perPage)

	data := homeData{
		Artists:      artistsPage,
		TotalArtists: totalArtists,
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
