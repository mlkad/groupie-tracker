package api

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

type ArtistFilter struct {
	CreationFrom int
	CreationTo   int

	AlbumFrom int
	AlbumTo   int

	MembersFrom int
	MembersTo   int

	Locations []string

	MembersExact int
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

	//фильтр по именам артистов
	for _, a := range c.artists {
		if strings.Contains(strings.ToLower(a.Name), q) {
			found = append(found, a)
			continue
		}

		matched := false
		for _, m := range a.Members {
			if strings.Contains(strings.ToLower(m), q) {
				matched = true
				break
			}
		}
		if matched {
			found = append(found, a)
			continue
		}

		//фильтр по дате создания группы
		if strings.Contains(strconv.Itoa(a.CreationDate), q) {
			found = append(found, a)
			continue
		}

		//фильтр по дате первого альбома
		if strings.Contains(a.FirstAlbum, q) {
			found = append(found, a)
			continue
		}

		//фильтр по локации
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

		//фильтр по дате концерта
		for _, d := range c.dates {
			if d.ID != a.ID {
				continue
			}
			for _, date := range d.Dates {
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

func firstAlbumInRange(date string, from, to int) bool {
	parts := strings.Split(date, "-")
	if len(parts) != 3 {
		return false
	}

	year, err := strconv.Atoi(parts[2])
	if err != nil {
		return false
	}

	return year >= from && year <= to
}

func (c *Cache) Filter(f ArtistFilter) []models.Artist {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := make([]models.Artist, 0)

	for _, artist := range c.artists {
		if artist.CreationDate < f.CreationFrom || artist.CreationDate > f.CreationTo {
			continue
		}

		if f.MembersExact > 0 {
			if len(artist.Members) != f.MembersExact {
				continue
			}
		} else if len(artist.Members) < f.MembersFrom || len(artist.Members) > f.MembersTo {
			continue
		}

		if !firstAlbumInRange(artist.FirstAlbum, f.AlbumFrom, f.AlbumTo) {
			continue
		}

		if len(f.Locations) != 0 {
			matched := false
			for _, loc := range f.Locations {
				if c.hasLocation(artist.ID, loc) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		res = append(res, artist)
	}

	return res
}

func (c *Cache) hasLocation(id int, query string) bool {
	query = strings.ToLower(query)

	for _, location := range c.locations {
		if location.ID != id {
			continue
		}

		for _, loc := range location.Locations {
			if strings.Contains(strings.ToLower(loc), query) {
				return true
			}
		}
		return false
	}
	return false
}

func (c *Cache) AllLocations() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	seen := make(map[string]bool)
	for _, l := range c.locations {
		for _, loc := range l.Locations {
			seen[loc] = true
		}
	}

	out := make([]string, 0, len(seen))
	for loc := range seen {
		out = append(out, loc)
	}
	sort.Strings(out)

	return out
}
