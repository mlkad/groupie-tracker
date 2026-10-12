package models

import (
	"encoding/json"
	"testing"
)

const artistsJSON = `[
  {
    "id": 1,
    "image": "https://groupietrackers.herokuapp.com/api/images/queen.jpeg",
    "name": "Queen",
    "members": ["Freddie Mercury", "Brian May", "John Daecon", "Roger Meddows-Taylor", "Mike Grose", "Barry Mitchell", "Doug Fogie"],
    "creationDate": 1970,
    "firstAlbum": "14-12-1973",
    "locations": "https://groupietrackers.herokuapp.com/api/locations/1",
    "concertDates": "https://groupietrackers.herokuapp.com/api/dates/1",
    "relations": "https://groupietrackers.herokuapp.com/api/relation/1"
  }
]`

func TestArtistUnmarshal(t *testing.T) {
	var artists []Artist
	if err := json.Unmarshal([]byte(artistsJSON), &artists); err != nil {
		t.Fatalf("не удалось разобрать artists: %v", err)
	}

	if len(artists) != 1 {
		t.Fatalf("ожидался 1 артист, получено %d", len(artists))
	}

	a := artists[0]

	if a.ID != 1 {
		t.Errorf("ID = %d, ожидалось 1", a.ID)
	}
	if a.Name != "Queen" {
		t.Errorf("Name = %q, ожидалось \"Queen\"", a.Name)
	}
	if a.CreationDate != 1970 {
		t.Errorf("CreationDate = %d, ожидалось 1970", a.CreationDate)
	}
	if a.FirstAlbum != "14-12-1973" {
		t.Errorf("FirstAlbum = %q, ожидалось \"14-12-1973\"", a.FirstAlbum)
	}
	if len(a.Members) != 7 {
		t.Errorf("len(Members) = %d, ожидалось 7", len(a.Members))
	}
	if len(a.Members) > 0 && a.Members[0] != "Freddie Mercury" {
		t.Errorf("Members[0] = %q, ожидалось \"Freddie Mercury\"", a.Members[0])
	}
	if a.Locations != "https://groupietrackers.herokuapp.com/api/locations/1" {
		t.Errorf("Locations = %q", a.Locations)
	}
	if a.ConcertDates != "https://groupietrackers.herokuapp.com/api/dates/1" {
		t.Errorf("ConcertDates = %q", a.ConcertDates)
	}
	if a.Relations != "https://groupietrackers.herokuapp.com/api/relation/1" {
		t.Errorf("Relations = %q", a.Relations)
	}
	if a.Image == "" {
		t.Error("Image пустой")
	}
}

const locationsJSON = `{
  "index": [
    {
      "id": 1,
      "locations": ["north_carolina-usa", "georgia-usa", "los_angeles-usa", "saitama-japan", "osaka-japan", "nagoya-japan", "penrose-new_zealand", "dunedin-new_zealand"],
      "dates": "https://groupietrackers.herokuapp.com/api/dates/1"
    }
  ]
}`

func TestLocationsUnmarshal(t *testing.T) {
	var resp LocationsResponse
	if err := json.Unmarshal([]byte(locationsJSON), &resp); err != nil {
		t.Fatalf("не удалось разобрать locations: %v", err)
	}

	if len(resp.Index) != 1 {
		t.Fatalf("len(Index) = %d, ожидался 1 (обёртка {\"index\": [...]} не разобралась?)", len(resp.Index))
	}

	l := resp.Index[0]

	if l.ID != 1 {
		t.Errorf("ID = %d, ожидалось 1", l.ID)
	}
	if len(l.Locations) != 8 {
		t.Errorf("len(Locations) = %d, ожидалось 8", len(l.Locations))
	}
	if len(l.Locations) > 0 && l.Locations[0] != "north_carolina-usa" {
		t.Errorf("Locations[0] = %q, ожидалось \"north_carolina-usa\"", l.Locations[0])
	}
	if l.Dates != "https://groupietrackers.herokuapp.com/api/dates/1" {
		t.Errorf("Dates = %q", l.Dates)
	}
}

const datesJSON = `{
  "index": [
    {
      "id": 1,
      "dates": ["*23-08-2019", "*22-08-2019", "*20-08-2019", "*26-01-2020", "*28-01-2020", "*30-01-2019", "*07-02-2020", "*10-02-2020"]
    }
  ]
}`

func TestDatesUnmarshal(t *testing.T) {
	var resp DatesResponse
	if err := json.Unmarshal([]byte(datesJSON), &resp); err != nil {
		t.Fatalf("не удалось разобрать dates: %v", err)
	}

	if len(resp.Index) != 1 {
		t.Fatalf("len(Index) = %d, ожидался 1", len(resp.Index))
	}

	d := resp.Index[0]

	if d.ID != 1 {
		t.Errorf("ID = %d, ожидалось 1", d.ID)
	}
	if len(d.Dates) != 8 {
		t.Errorf("len(Dates) = %d, ожидалось 8", len(d.Dates))
	}
	if len(d.Dates) > 0 && d.Dates[0] != "*23-08-2019" {
		t.Errorf("Dates[0] = %q, ожидалось \"*23-08-2019\"", d.Dates[0])
	}
}

const relationJSON = `{
  "index": [
    {
      "id": 1,
      "datesLocations": {
        "dunedin-new_zealand": ["10-02-2020"],
        "georgia-usa": ["22-08-2019"],
        "los_angeles-usa": ["20-08-2019"],
        "nagoya-japan": ["30-01-2019"],
        "north_carolina-usa": ["23-08-2019"],
        "osaka-japan": ["28-01-2020"],
        "penrose-new_zealand": ["07-02-2020"],
        "saitama-japan": ["26-01-2020"]
      }
    },
    {
      "id": 2,
      "datesLocations": {
        "noumea-new_caledonia": ["15-11-2019"],
        "papeete-french_polynesia": ["16-11-2019"],
        "playa_del_carmen-mexico": ["05-12-2019", "06-12-2019", "07-12-2019", "08-12-2019", "09-12-2019"]
      }
    }
  ]
}`

func TestRelationUnmarshal(t *testing.T) {
	var resp RelationResponse
	if err := json.Unmarshal([]byte(relationJSON), &resp); err != nil {
		t.Fatalf("не удалось разобрать relation: %v", err)
	}

	if len(resp.Index) != 2 {
		t.Fatalf("len(Index) = %d, ожидалось 2", len(resp.Index))
	}

	r := resp.Index[0]

	if r.ID != 1 {
		t.Errorf("ID = %d, ожидалось 1", r.ID)
	}

	if len(r.DatesLocations) == 0 {
		t.Fatal("DatesLocations пустая — проверь тег json:\"datesLocations\"")
	}
	if len(r.DatesLocations) != 8 {
		t.Errorf("len(DatesLocations) = %d, ожидалось 8", len(r.DatesLocations))
	}

	got, ok := r.DatesLocations["georgia-usa"]
	if !ok {
		t.Fatal("ключ \"georgia-usa\" не найден")
	}
	if len(got) != 1 || got[0] != "22-08-2019" {
		t.Errorf("DatesLocations[\"georgia-usa\"] = %v, ожидалось [22-08-2019]", got)
	}

	second := resp.Index[1].DatesLocations["playa_del_carmen-mexico"]
	if len(second) != 5 {
		t.Errorf("len(DatesLocations[\"playa_del_carmen-mexico\"]) = %d, ожидалось 5", len(second))
	}
}
