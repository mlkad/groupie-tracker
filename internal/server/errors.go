package server

import (
	"log"
	"net/http"
)

type errorData struct {
	Status  int
	Title   string
	Message string
	Query   string
}

func (s *Server) clientError(w http.ResponseWriter, status int) {
	s.render(w, status, "error.html", errorData{Status: status, Title: http.StatusText(status), Message: "something went wrong"})
}

func (s *Server) serverError(w http.ResponseWriter, err error) {
	log.Printf("server error: %v", err)
	s.clientError(w, http.StatusInternalServerError)
}
