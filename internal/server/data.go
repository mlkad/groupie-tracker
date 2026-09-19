package server

import "github.com/mlkad/groupie-tracker/internal/models"

type homeData struct {
	Artists []models.Artist
	Query   string

	CreationMin, CreationMax int
	AlbumMin, AlbumMax       int
	MembersMin, MembersMax   int

	CreationFrom, CreationTo string
	AlbumFrom, AlbumTo       string
	MembersFrom, MembersTo   string
	MembersExact             string

	Sort string

	Locations []string
	Selected  map[string]bool
}

type concert struct {
	Location string
	Dates    []string
}

type artistData struct {
	Artist   models.Artist
	Concerts []concert
	Query    string
}

type searchData struct {
	Artists []models.Artist
	Query   string
}

type suggestion struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
