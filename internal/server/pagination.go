package server

import "github.com/mlkad/groupie-tracker/internal/models"

func paginate(items []models.Artist, page, perPage int) ([]models.Artist, int) {
	if perPage <= 0 {
		return items, 1
	}

	totalPages := (len(items) + perPage - 1) / perPage // 12 показ: x = (52 + 12 - 1) / 12 = 5 стр

	if totalPages == 0 {
		totalPages = 1
	}

	if page < 1 {
		page = 1
	}

	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * perPage
	end := start + perPage

	if end > len(items) {
		end = len(items)
	}

	return items[start:end], totalPages
}
