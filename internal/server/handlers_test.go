package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mlkad/groupie-tracker/internal/api"
)

func fakeAPI(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	var srv *httptest.Server

	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"artists":   "` + srv.URL + `/api/artists",
			"locations": "` + srv.URL + `/api/locations",
			"dates":     "` + srv.URL + `/api/dates",
			"relation":  "` + srv.URL + `/api/relation"
		}`))
	})

	mux.HandleFunc("/api/artists", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[
			{"id":1,"image":"queen.jpeg","name":"Queen",
			 "members":["Freddie Mercury","Brian May"],
			 "creationDate":1970,"firstAlbum":"14-12-1973",
			 "locations":"x","concertDates":"y","relations":"z"},
			{"id":2,"image":"soja.jpeg","name":"SOJA",
			 "members":["Jacob Hemphill"],
			 "creationDate":1997,"firstAlbum":"28-07-2002",
			 "locations":"x","concertDates":"y","relations":"z"}
		]`))
	})

	mux.HandleFunc("/api/locations", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"index":[{"id":1,"locations":["georgia-usa"],"dates":"x"}]}`))
	})

	mux.HandleFunc("/api/dates", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"index":[{"id":1,"dates":["*22-08-2019"]}]}`))
	})

	mux.HandleFunc("/api/relation", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"index":[
			{"id":1,"datesLocations":{
				"north_carolina-usa":["*23-08-2019"],
				"georgia-usa":["22-08-2019"],
				"dunedin-new_zealand":["10-02-2020"]}},
			{"id":2,"datesLocations":{}}
		]}`))
	})

	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(wd) })

	apiSrv := fakeAPI(t)

	client := api.NewClientWithBaseURL(apiSrv.URL + "/api")
	cache := api.NewCache(client)
	if err := cache.Load(context.Background()); err != nil {
		t.Fatalf("cache.Load: %v", err)
	}

	srv, err := New(cache)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return srv.Routes()
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHomeOK(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/")

	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидалось 200", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Queen") {
		t.Error("на главной нет Queen")
	}
	if !strings.Contains(body, "SOJA") {
		t.Error("на главной нет SOJA")
	}
	if !strings.Contains(body, "<!doctype html>") {
		t.Error("ответ не похож на HTML-страницу")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, ожидался text/html", ct)
	}
}

func TestArtistOK(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/artist/1")

	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидалось 200", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{"Queen", "Freddie Mercury", "1970", "14-12-1973"} {
		if !strings.Contains(body, want) {
			t.Errorf("на странице артиста нет %q", want)
		}
	}
}

func TestArtistFormatsLocationsAndDates(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/artist/1").Body.String()

	if !strings.Contains(body, "North Carolina, Usa") {
		t.Error("локация не отформатирована (ожидалось \"North Carolina, Usa\")")
	}
	if strings.Contains(body, "north_carolina-usa") {
		t.Error("сырой ключ локации попал на страницу")
	}
	if strings.Contains(body, "*23-08-2019") {
		t.Error("звёздочка не срезана с даты")
	}
	if !strings.Contains(body, "23-08-2019") {
		t.Error("дата отсутствует на странице")
	}
}

func TestArtistConcertsSortedStably(t *testing.T) {
	h := newTestServer(t)

	var first []string
	for i := 0; i < 5; i++ {
		body := get(t, h, "/artist/1").Body.String()
		got := locationsFrom(body)

		if i == 0 {
			first = got
			continue
		}
		if len(got) != len(first) {
			t.Fatalf("запрос %d вернул %d локаций, в первом было %d", i, len(got), len(first))
		}
		for j := range got {
			if got[j] != first[j] {
				t.Fatalf("порядок локаций нестабилен: запрос %d дал %v, первый %v", i, got, first)
			}
		}
	}

	for i := 1; i < len(first); i++ {
		if first[i-1] > first[i] {
			t.Errorf("локации не отсортированы: %q идёт перед %q", first[i-1], first[i])
		}
	}
}

func locationsFrom(body string) []string {
	var out []string
	const open = `<td class="loc">`
	rest := body
	for {
		i := strings.Index(rest, open)
		if i < 0 {
			return out
		}
		rest = rest[i+len(open):]
		j := strings.Index(rest, "</td>")
		if j < 0 {
			return out
		}
		out = append(out, rest[:j])
		rest = rest[j:]
	}
}

func TestArtistWithoutConcerts(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/artist/2")

	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидалось 200 (артист есть, концертов нет)", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No concerts listed") {
		t.Error("нет сообщения о пустом списке концертов")
	}
}

func TestArtistNotFound(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/artist/999")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("статус = %d, ожидалось 404", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "404") {
		t.Error("страница ошибки не содержит код 404")
	}
	if !strings.Contains(body, "<!doctype html>") {
		t.Error("страница 404 не HTML")
	}
}

func TestArtistBadID(t *testing.T) {
	h := newTestServer(t)

	for _, path := range []string{"/artist/abc", "/artist/1.5", "/artist/%20"} {
		rec := get(t, h, path)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("GET %s: статус = %d, ожидалось 400", path, rec.Code)
		}
	}
}

func TestArtistNegativeID(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/artist/-5")

	if rec.Code != http.StatusNotFound {
		t.Errorf("статус = %d, ожидалось 404", rec.Code)
	}
}

func TestUnknownPath(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/no-such-page")

	if rec.Code != http.StatusNotFound {
		t.Errorf("статус = %d, ожидалось 404", rec.Code)
	}
}

func TestSearchByName(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/search?q=queen")

	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидалось 200", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Queen") {
		t.Error("Queen не найден по имени")
	}
	if strings.Contains(body, "SOJA") {
		t.Error("SOJA не должен попасть в результаты по запросу \"queen\"")
	}
}

func TestSearchByMember(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/search?q=freddie").Body.String()

	if !strings.Contains(body, "Queen") {
		t.Error("Queen не найден по имени участника")
	}
}

func TestSearchByYear(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/search?q=1997").Body.String()

	if !strings.Contains(body, "SOJA") {
		t.Error("SOJA не найден по году создания")
	}
}

func TestSearchNoResults(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/search?q=zzzzz")

	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидалось 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Nothing matched") {
		t.Error("нет сообщения об отсутствии результатов")
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/search")

	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидалось 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "Queen") || strings.Contains(body, "SOJA") {
		t.Error("пустой запрос не должен возвращать артистов")
	}
}

func TestSearchQueryEchoedInInput(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/search?q=queen").Body.String()

	if !strings.Contains(body, `value="queen"`) {
		t.Error("запрос не подставлен обратно в поле поиска")
	}
}

func TestSearchEscapesHTML(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/search?q=%3Cscript%3Ealert(1)%3C/script%3E").Body.String()

	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Error("пользовательский ввод не экранирован — XSS")
	}
}

func TestSuggestJSON(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/suggest?q=que")

	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидалось 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, ожидался application/json", ct)
	}

	var got []suggestion
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("ответ не разбирается как JSON: %v (тело: %s)", err, rec.Body.String())
	}

	if len(got) != 1 {
		t.Fatalf("len = %d, ожидалась 1 подсказка", len(got))
	}
	if got[0].Name != "Queen" {
		t.Errorf("Name = %q, ожидалось \"Queen\"", got[0].Name)
	}
	if got[0].ID != 1 {
		t.Errorf("ID = %d, ожидалось 1", got[0].ID)
	}
}

func TestSuggestEmptyIsArrayNotNull(t *testing.T) {
	h := newTestServer(t)

	for _, path := range []string{"/suggest?q=zzzzz", "/suggest?q=", "/suggest"} {
		rec := get(t, h, path)
		body := strings.TrimSpace(rec.Body.String())

		if body != "[]" {
			t.Errorf("GET %s: тело = %q, ожидалось \"[]\" (null сломает JS)", path, body)
		}
	}
}

func TestSuggestLimit(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/suggest?q=e")

	var got []suggestion
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if len(got) > 8 {
		t.Errorf("вернулось %d подсказок, максимум 8", len(got))
	}
}

func TestStaticFiles(t *testing.T) {
	h := newTestServer(t)

	cases := []struct {
		path string
		ct   string
	}{
		{"/static/css/main.css", "text/css"},
		{"/static/js/search.js", "javascript"},
	}

	for _, c := range cases {
		rec := get(t, h, c.path)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: статус = %d, ожидалось 200", c.path, rec.Code)
			continue
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, c.ct) {
			t.Errorf("GET %s: Content-Type = %q, ожидалось содержащее %q", c.path, ct, c.ct)
		}
	}
}

func TestStaticDirectoryTraversal(t *testing.T) {
	h := newTestServer(t)
	rec := get(t, h, "/static/../../go.mod")

	if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "module ") {
		t.Error("выход за пределы web/static — файл go.mod отдан наружу")
	}
}

func TestRecoverPanic(t *testing.T) {
	panicky := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	recoverPanic(panicky).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("статус = %d, ожидалось 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Error("текст паники утёк в ответ клиенту")
	}
}

func TestRecoverPanicKeepsServing(t *testing.T) {
	h := newTestServer(t)

	if rec := get(t, h, "/artist/abc"); rec.Code != http.StatusBadRequest {
		t.Fatalf("подготовка: статус = %d", rec.Code)
	}
	if rec := get(t, h, "/"); rec.Code != http.StatusOK {
		t.Errorf("после ошибочного запроса главная вернула %d", rec.Code)
	}
}

func TestPostNotAllowed(t *testing.T) {
	h := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Error("POST / вернул 200, ожидалось отклонение")
	}
}
