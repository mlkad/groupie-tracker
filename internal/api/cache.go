package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/mlkad/groupie-tracker/internal/models"
)

//чтоб не ходить в хероку на каждый запрос

type Cache struct {
	mu        sync.RWMutex
	client    *Client
	artists   []models.Artist
	locations []models.Location
	dates     []models.Date
	relations []models.Relation
}

func NewCache(client *Client) *Cache {
	return &Cache{client: client}
}

func (c *Cache) Load(ctx context.Context) error {
	if err := c.client.Discover(ctx); err != nil {
		return fmt.Errorf("discovering api: %w", err)
	}

	artists, err := c.client.FetchArtists(ctx)
	if err != nil {
		return fmt.Errorf("loading artists: %w", err)
	}

	locations, err := c.client.FetchLocations(ctx)
	if err != nil {
		return fmt.Errorf("loading locations: %w", err)
	}

	dates, err := c.client.FetchDates(ctx)
	if err != nil {
		return fmt.Errorf("loading dates: %w", err)
	}

	relations, err := c.client.FetchRelations(ctx)
	if err != nil {
		return fmt.Errorf("loading relations: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.artists = artists
	c.locations = locations
	c.dates = dates
	c.relations = relations

	return nil
}

// возвращает всех артистов из кэша
func (c *Cache) Artists() []models.Artist {
	c.mu.RLock()         //я сейчас читаю данные, другие читатели пусть читают тоже, а вот писателю надо подождать
	defer c.mu.RUnlock() //открыть замок
	return c.artists
}

func (c *Cache) Relation(id int) (models.Relation, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, r := range c.relations {
		if r.ID == id {
			return r, nil
		}
	}
	return models.Relation{}, ErrNotFound
}

var ErrNotFound = errors.New("artist not found")

func (c *Cache) Artist(id int) (models.Artist, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, a := range c.artists {
		if a.ID == id {
			return a, nil
		}
	}
	return models.Artist{}, ErrNotFound
}

func (c *Cache) Search(q string) []models.Artist {
	c.mu.RLock()
	defer c.mu.RUnlock()

	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return nil
	}

	var found []models.Artist

	for _, a := range c.artists {
		if strings.Contains(strings.ToLower(a.Name), q) { //по имени
			found = append(found, a)
			continue
		}

		matched := false
		for _, m := range a.Members { //по участникам
			if strings.Contains(strings.ToLower(m), q) {
				matched = true
				break
			}
		}
		if matched {
			found = append(found, a)
			continue
		}

		if strings.Contains(strconv.Itoa(a.CreationDate), q) { //дата
			found = append(found, a)
			continue
		}

		if strings.Contains(a.FirstAlbum, q) {
			found = append(found, a)
			continue
		}

		for _, l := range c.locations {
			if l.ID != a.ID {
				continue
			}
			for _, loc := range l.Locations {
				if strings.Contains(loc, q) {
					matched = true
					break
				}
			}
			break
		}
		if matched {
			found = append(found, a)
			continue
		}

		for _, d := range c.dates {
			if d.ID != a.ID {
				continue
			}
			for _, date := range d.Dates { //
				if strings.Contains(date, q) {
					matched = true
					break
				}
			}
			break
		}
		if matched {
			found = append(found, a)
			continue
		}
	}

	return found
}
