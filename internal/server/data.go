package server

import "github.com/mlkad/groupie-tracker/internal/models"

type homeData struct {
	Artists []models.Artist
	Query   string
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
