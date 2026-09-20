# Groupie Tracker

A website about music bands: line-up, creation year, first album and concert
geography. Data comes from the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api).

The backend is written in Go using the standard library only.

**Live demo:** https://groupie-tracker-t3is.onrender.com

![Groupie Tracker](docs/screenshot.png)

## Running

```bash
go run ./cmd/web
```

The site opens at http://localhost:8080

To change the port:

```bash
go run ./cmd/web -addr :3000
```

Run it from the project root — template and static paths are relative.

## Tests

```bash
go test ./...
go test -race ./...
```

53 tests, none of which call the external API — a local `httptest.Server`
is used instead.

## API Structure

The application consumes the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api) which consists of four main endpoints:

- **artists** — band/artist information (names, image, creation year, first album date, members)
- **locations** — last and upcoming concert locations
- **dates** — last and upcoming concert dates
- **relation** — links artists with their concert dates and locations

## Routes

| route | purpose |
|---|---|
| `GET /` | artist list; accepts the same filter parameters as `/filter` |
| `GET /filter` | the list again, the URL the filter form submits to |
| `GET /artist/{id}` | one artist with the concert table |
| `GET /search?q=` | full search results page |
| `GET /suggest?q=` | JSON suggestions for the search box (max 8) |
| `GET /static/…` | CSS and JS |

## Features

- **Home** — grid of all artists with data visualization
- **Artist page** — photo, members, creation year, first album,
  and a "city → concert dates" table
- **Search** by band name with live suggestions — real-time client-server event: while typing, 
  the browser requests `/suggest?q=...` and shows matches without reloading the page 
  (demonstrates client-initiated request-response communication)
- **Advanced filters** — filter artists by:
  - Creation year range (from-to sliders paired with number inputs)
  - First album date range (from-to sliders paired with number inputs)
  - Number of band members (range slider, plus an optional "exact number" field)
  - Concert locations (searchable checklist, collapsible, "show more")
- **Sorting** — by name (A→Z, Z→A), creation year (oldest, newest)
  and member count (fewest, most)
- **Pagination** — 8 artists per page, with a sliding window of page numbers
  that keeps the control row a fixed width
- **Responsive design** — collapsible filter panel on mobile, full sidebar on desktop
- **Debounced filter application** — filters update results asynchronously without page reload

## Structure

```
cmd/web/          entry point: flags, dependency wiring, graceful shutdown
internal/
  models/         API data types
  api/
    client.go     HTTP client: discovery, four endpoints, timeouts
    cache.go      in-memory storage, search and filtering
  server/
    server.go     routes, template parsing, rendering
    handlers.go   page handlers
    data.go       view models passed to templates
    pagination.go slicing a result set into pages
    errors.go     error pages
    middleware.go panic recovery
    format.go     location and date formatting
web/
  templates/      HTML templates
  static/         CSS and JS
```

## How it works

On startup the program requests the API root document, reads the addresses of
the four endpoints, loads `artists`, `locations`, `dates` and `relation`, and
stores them in memory. After that the site serves everything from the cache —
the external API is never called again, so pages load instantly and stay
available even if herokuapp goes down.

Data from `relation` fills the concert table on the artist page;
`locations` and `dates` are used by the search.

## How filtering works

Filtering happens on the server. The browser collects the form into a query
string and asks `/filter?...`; the handler parses the parameters into an
`api.ArtistFilter`, `cache.Filter` walks the cached artists once and keeps
those matching every active criterion, then the result is sorted and sliced
into a page.

Missing or unparsable parameters fall back to neutral bounds (`0` and
`math.MaxInt`), so a filter the user never touched never narrows the result.
The "exact number of members" field takes precedence over the members range
when it is filled in.

Location matching is a substring test, which is what the subject's hint about
nested places asks for: selecting `washington-usa` also matches an artist whose
location is `seattle-washington-usa`.

The same handler serves `/` and `/filter`, so a filtered URL can be opened,
bookmarked or shared directly — the page renders with the filters applied and
the form fields already filled in.

The JavaScript layer only makes this asynchronous: it debounces rapid slider
moves by 300 ms, aborts a superseded request with an `AbortController` so a
slow answer cannot overwrite a newer one, and swaps in the `#results` block
from the response. Filtering and sorting degrade gracefully — with JavaScript
disabled the form still submits as a plain GET. The page controls are the one
part that needs JavaScript, since the page numbers are rendered on the client.

## Error handling

| situation | response |
|---|---|
| `/artist/abc` | 400 |
| `/artist/9999` | 404 |
| unknown path | 404 |
| POST, PUT, DELETE | 405 |
| panic in a handler | 500, server keeps running |

The server shuts down gracefully on `Ctrl+C` or `SIGTERM`: it stops accepting
new requests and waits for the in-flight ones to finish.

## Requirements

Go 1.25.5 or newer, as declared in `go.mod`. The routing relies on method and
path patterns in `net/http` (`GET /artist/{id}`), which need Go 1.22 at minimum.

No external dependencies — the standard library only.
