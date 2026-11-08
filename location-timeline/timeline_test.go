package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sander-skjulsvik/tools/libs/immich"
)

const (
	me       = "11111111-1111-1111-1111-111111111111"
	namesake = "22222222-2222-2222-2222-222222222222"
	friend   = "33333333-3333-3333-3333-333333333333"
)

type place struct {
	lat, lon      float64
	city, country string
}

var (
	bergen = place{60.39, 5.32, "Bergen", "Norway"}
	askoy  = place{60.41, 5.20, "Askøy", "Norway"}
	oslo   = place{59.91, 10.75, "Oslo", "Norway"}
	madrid = place{40.42, -3.70, "Madrid", "Spain"}
)

var config = &ImmichConfig{Cameras: []Camera{{Make: "Apple", Model: "iPhone 15"}, {Make: "FUJIFILM"}}, People: []string{"Me"}}

// photo is taken at a UTC time given as 2006-01-02T15:04, in a UTC+1 place.
func photo(t *testing.T, at string, where place, make, model string, people ...PersonRef) Photo {
	t.Helper()
	taken, err := time.Parse("2006-01-02T15:04", at)
	if err != nil {
		t.Fatal(err)
	}
	return Photo{
		ID: at, Taken: taken, LocalTime: taken.Add(time.Hour).Format(localLayout),
		Latitude: where.lat, Longitude: where.lon, City: where.city, Country: where.country,
		Make: make, Model: model, People: people,
	}
}

func mine(t *testing.T, at string, where place) Photo {
	return photo(t, at, where, "Apple", "iPhone 15")
}

func TestOnlyOwnCamerasAndOwnFaceCount(t *testing.T) {
	photos := []Photo{
		mine(t, "2025-01-01T10:00", bergen),
		photo(t, "2025-01-01T11:00", madrid, "samsung", "SM-S911B"),                                // forwarded
		photo(t, "2025-01-01T12:00", oslo, "samsung", "SM-S911B", PersonRef{me, "Me"}),             // a friend's photo of me
		photo(t, "2025-01-01T13:00", oslo, "FUJIFILM", "X100V", PersonRef{friend, "Friend"}),       // any Fujifilm
		photo(t, "2025-01-01T14:00", madrid, "Apple", "iPhone 12", PersonRef{friend, "Friend"}),    // someone else's iPhone
		photo(t, "2025-01-01T15:00", oslo, "Apple", "iPhone 15", PersonRef{me, "Me"}, PersonRef{}), // both
	}
	points := immichPoints(photos, config)
	var got []string
	for _, p := range points {
		got = append(got, p.ID+" "+strings.Join(p.Evidence, "+"))
	}
	want := []string{
		"2025-01-01T10:00 camera", "2025-01-01T12:00 person", "2025-01-01T13:00 camera", "2025-01-01T15:00 camera+person",
	}
	if strings.Join(got, ", ") != strings.Join(want, ", ") {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPeopleMatchByIDToo(t *testing.T) {
	byID := &ImmichConfig{People: []string{me}}
	if len(immichPoints([]Photo{photo(t, "2025-01-01T10:00", oslo, "", "", PersonRef{me, ""})}, byID)) != 1 {
		t.Error("an unnamed person did not match by ID")
	}
}

func TestNamesakesAreRefused(t *testing.T) {
	photos := []Photo{
		photo(t, "2025-01-01T10:00", oslo, "", "", PersonRef{me, "Me"}),
		photo(t, "2025-01-01T11:00", oslo, "", "", PersonRef{namesake, "me"}),
	}
	err := config.checkPeople(photos)
	if err == nil || !strings.Contains(err.Error(), namesake) {
		t.Errorf("got %v", err)
	}
}

func TestPhotoFromAsset(t *testing.T) {
	decode := func(asset map[string]any) (Photo, bool) {
		raw, _ := json.Marshal(asset)
		photo, ok, err := photoFromAsset(raw)
		if err != nil {
			t.Fatal(err)
		}
		return photo, ok
	}
	photo, ok := decode(map[string]any{
		"id": "a", "fileCreatedAt": "2025-06-01T10:00:00.000Z", "localDateTime": "2025-06-01T12:00:00.000Z",
		"exifInfo": map[string]any{
			"dateTimeOriginal": "2025-06-01T09:59:58.000+00:00", "timeZone": "Europe/Oslo",
			"latitude": 60.39, "longitude": 5.32, "city": "Bergen", "country": "Norway", "make": "Apple", "model": "iPhone 15",
		},
		"people": []map[string]any{{"id": me, "name": "Me", "birthDate": nil}},
	})
	if !ok || photo.LocalTime != "2025-06-01T12:00:00" || !photo.Taken.Equal(time.Date(2025, 6, 1, 9, 59, 58, 0, time.UTC)) ||
		len(photo.People) != 1 || photo.People[0].Name != "Me" || photo.Model != "iPhone 15" {
		t.Errorf("got %+v, %v", photo, ok)
	}
	for _, exif := range []map[string]any{
		{"latitude": nil, "longitude": nil},
		{"latitude": 0, "longitude": 0},
		{},
	} {
		if _, ok := decode(map[string]any{"id": "b", "fileCreatedAt": "2025-06-01T10:00:00.000Z",
			"localDateTime": "2025-06-01T12:00:00.000Z", "exifInfo": exif}); ok {
			t.Errorf("kept a photo without a location: %v", exif)
		}
	}
}

func cityOptions() StayOptions {
	return StayOptions{Level: "city", RadiusKm: 25, MaxGap: 72 * time.Hour}
}

func points(t *testing.T, photos ...Photo) []Point {
	t.Helper()
	return immichPoints(photos, config)
}

func describe(entries []Entry) string {
	var parts []string
	for _, entry := range entries {
		if entry.Kind == "gap" {
			parts = append(parts, "gap")
		} else {
			parts = append(parts, entry.Place+" "+strconv.Itoa(entry.Points))
		}
	}
	return strings.Join(parts, " | ")
}

func TestStaysSplitOnDistanceAndShowGaps(t *testing.T) {
	trip := points(t,
		mine(t, "2025-03-01T10:00", bergen),
		mine(t, "2025-03-01T18:00", askoy), // within 25 km
		mine(t, "2025-03-02T10:00", bergen),
		mine(t, "2025-03-02T20:00", oslo),
		mine(t, "2025-03-03T09:00", oslo),
		mine(t, "2025-03-04T09:00", bergen),
		mine(t, "2025-03-20T09:00", bergen), // 16 days later
	)
	got := describe(Stays(trip, cityOptions()))
	want := "Bergen, Norway 3 | Oslo, Norway 2 | Bergen, Norway 1 | gap | Bergen, Norway 1"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	first := Stays(trip, cityOptions())[0]
	if first.Days != 2 || len(first.Places) != 2 || first.Places[1].Place != "Askøy, Norway" {
		t.Errorf("got %+v", first)
	}
}

func TestCountryLevelJoinsCities(t *testing.T) {
	trip := points(t,
		mine(t, "2025-03-01T10:00", bergen),
		mine(t, "2025-03-02T10:00", oslo),
		mine(t, "2025-03-03T10:00", madrid),
		mine(t, "2025-03-04T10:00", oslo),
	)
	got := describe(Stays(trip, StayOptions{Level: "country", MaxGap: 72 * time.Hour}))
	if want := "Norway 2 | Spain 1 | Norway 1"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestOnADateInAGapShowsTheGap(t *testing.T) {
	entries := Stays(points(t,
		mine(t, "2025-03-01T10:00", bergen),
		mine(t, "2025-03-20T10:00", madrid),
	), cityOptions())
	if got := describe(InRange(entries, Range{"2025-03-10", "2025-03-10"})); got != "gap" {
		t.Errorf("got %s", got)
	}
	if got := describe(InRange(entries, Range{"2025-03-20", ""})); got != "Madrid, Spain 1" {
		t.Errorf("got %s", got)
	}
	if got := describe(InRange(entries, Range{"", "2025-03-01"})); got != "Bergen, Norway 1" {
		t.Errorf("got %s", got)
	}
	if got := describe(InRange(entries, Range{"2025-03-01", "2025-03-20"})); got != "Bergen, Norway 1 | gap | Madrid, Spain 1" {
		t.Errorf("got %s", got)
	}
}

func TestDaysUseLocalDatesAndKeepEmptyDays(t *testing.T) {
	trip := points(t,
		mine(t, "2025-03-01T23:30", oslo), // 00:30 on the 2nd, local time
		mine(t, "2025-03-02T10:00", oslo),
		mine(t, "2025-03-02T15:00", bergen),
		mine(t, "2025-03-05T10:00", bergen),
	)
	days := Days(trip, Range{})
	var got []string
	for _, day := range days {
		var places []string
		for _, place := range day.Places {
			places = append(places, place.City+" "+strconv.Itoa(place.Points)+" "+place.First)
		}
		got = append(got, day.Date+": "+strings.Join(places, ", "))
	}
	want := []string{"2025-03-02: Oslo 2 00:30, Bergen 1 16:00", "2025-03-03: ", "2025-03-04: ", "2025-03-05: Bergen 1 11:00"}
	if strings.Join(got, "; ") != strings.Join(want, "; ") {
		t.Errorf("got  %v\nwant %v", got, want)
	}
	if len(Days(trip, Range{"2025-02-27", "2025-02-28"})) != 2 {
		t.Error("an explicit range without photos should still list its days")
	}
}

func TestCountriesReportCoverage(t *testing.T) {
	trip := points(t,
		mine(t, "2025-03-01T10:00", oslo),
		mine(t, "2025-03-02T10:00", madrid),
		mine(t, "2025-03-02T20:00", oslo),
		mine(t, "2025-03-06T10:00", oslo),
		mine(t, "2025-04-06T10:00", madrid), // outside the range
	)
	report := Countries(trip, Range{"2025-03-01", "2025-03-10"})
	coverage := report.Coverage
	if coverage.Days != 10 || coverage.DaysWithPoints != 3 || coverage.LongestGapDays != 4 || coverage.LongestGapSince != "2025-03-07" {
		t.Errorf("got %+v", coverage)
	}
	if len(report.Countries) != 2 || report.Countries[0] != (CountryVisit{"Norway", "2025-03-01", "2025-03-06", 3}) ||
		report.Countries[1] != (CountryVisit{"Spain", "2025-03-02", "2025-03-02", 1}) {
		t.Errorf("got %+v", report.Countries)
	}
}

func TestParseDate(t *testing.T) {
	today := time.Date(2026, 11, 8, 15, 0, 0, 0, time.Local)
	for value, want := range map[string]string{
		"2025-01-31": "2025-01-31", "30d": "2026-10-09", "2w": "2026-10-25", "6m": "2026-05-08", "1y": "2025-11-08", "": "",
	} {
		if got, err := ParseDate(value, today); err != nil || got != want {
			t.Errorf("%q: got %q, %v, want %q", value, got, err, want)
		}
	}
	for _, value := range []string{"2025-13-01", "6", "6x", "-1d", "2025-01"} {
		if _, err := ParseDate(value, today); err == nil {
			t.Errorf("%q: expected an error", value)
		}
	}
}

// fakeImmich serves geotagged and untagged assets, a page at a time.
func fakeImmich(t *testing.T, assets []map[string]any) (*httptest.Server, *[]string) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		var body struct {
			Page       int  `json:"page"`
			WithExif   bool `json:"withExif"`
			WithPeople bool `json:"withPeople"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if !body.WithExif || !body.WithPeople {
			t.Errorf("unexpected body %+v", body)
		}
		const size = 2
		start := min((body.Page-1)*size, len(assets))
		end := min(start+size, len(assets))
		var next *string
		if end < len(assets) {
			value := strconv.Itoa(body.Page + 1)
			next = &value
		}
		json.NewEncoder(w).Encode(map[string]any{"assets": map[string]any{"items": assets[start:end], "nextPage": next}})
	}))
	t.Cleanup(server.Close)
	return server, &calls
}

func asset(id, utc, local string, where *place, make string, people ...string) map[string]any {
	exif := map[string]any{"make": make, "model": "iPhone 15", "latitude": nil, "longitude": nil}
	if where != nil {
		exif["latitude"], exif["longitude"], exif["city"], exif["country"] = where.lat, where.lon, where.city, where.country
	}
	var refs []map[string]string
	for _, person := range people {
		refs = append(refs, map[string]string{"id": me, "name": person})
	}
	return map[string]any{"id": id, "fileCreatedAt": utc, "localDateTime": local, "exifInfo": exif, "people": refs}
}

func TestSyncThenReport(t *testing.T) {
	server, calls := fakeImmich(t, []map[string]any{
		asset("a", "2025-03-01T10:00:00.000Z", "2025-03-01T11:00:00.000Z", &oslo, "Apple"),
		asset("b", "2025-03-01T11:00:00.000Z", "2025-03-01T12:00:00.000Z", nil, "Apple"),
		asset("c", "2025-03-03T10:00:00.000Z", "2025-03-03T11:00:00.000Z", &madrid, "Apple"),
		asset("d", "2025-03-04T10:00:00.000Z", "2025-03-04T11:00:00.000Z", &bergen, "Canon"),
		asset("e", "2025-03-05T10:00:00.000Z", "2025-03-05T11:00:00.000Z", &bergen, "Canon", "Me"),
	})
	dir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("IMMICH_API_KEY", "key")
	if err := run([]string{"-data-dir", dir, "-url", server.URL, "sync"}, &bytes.Buffer{}); err == nil {
		t.Error("sync without a source should be refused")
	}
	if err := run([]string{"-data-dir", dir, "-url", server.URL, "sync", "immich"}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, call := range *calls {
		if call != "POST /api/search/metadata" {
			t.Errorf("unexpected request %s", call)
		}
	}
	data, err := LoadImmich(dir)
	if err != nil || data.Assets != 5 || len(data.Photos) != 4 {
		t.Fatalf("got %+v, %v", data, err)
	}

	var out bytes.Buffer
	if err := run([]string{"-data-dir", dir, "stays"}, &out); err == nil || !strings.Contains(err.Error(), "config.json") {
		t.Errorf("expected a missing config error, got %v", err)
	}
	if err := run([]string{"-data-dir", dir, "cameras"}, &out); err != nil || !strings.Contains(out.String(), "Canon") {
		t.Errorf("cameras: %v\n%s", err, out.String())
	}

	if err := writeJSON(filepath.Join(dir, configFile), Config{Immich: &ImmichConfig{Cameras: []Camera{{Make: "apple"}}, People: []string{"me"}}}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run([]string{"-data-dir", dir, "countries", "-since", "2025-03-01", "-until", "2025-03-05"}, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"3 of 5 days have photos", "Norway", "Spain"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("countries output lacks %q:\n%s", want, out.String())
		}
	}

	out.Reset()
	if err := run([]string{"-data-dir", dir, "stays", "-level", "country"}, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Norway", "Spain", "1 photo "} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("stays output lacks %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "2025-03-04") {
		t.Errorf("a Canon photo without me was counted:\n%s", out.String())
	}

	out.Reset()
	if err := run([]string{"-data-dir", dir, "points", "-on", "2025-03-05"}, &out); err != nil {
		t.Fatal(err)
	}
	var exported []Point
	if err := json.Unmarshal(out.Bytes(), &exported); err != nil || len(exported) != 1 || exported[0].ID != "e" {
		t.Errorf("got %+v, %v", exported, err)
	}
}

func TestOnlyMetadataSearchIsAllowed(t *testing.T) {
	server, calls := fakeImmich(t, nil)
	client, err := immich.NewClient(server.URL, "key", allowedRequests)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/assets", "/api/search/smart", "/api/map/markers"} {
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
			if _, _, err := client.Do(method, path, nil, nil); err == nil {
				t.Errorf("%s %s was allowed", method, path)
			}
		}
	}
	if len(*calls) != 0 {
		t.Errorf("requests reached the server: %v", *calls)
	}
}

// fakeSource stands in for a second source, such as Google Timeline.
type fakeSource struct {
	name   string
	points []Point
}

var _ Source = fakeSource{}

func (s fakeSource) Name() string             { return s.name }
func (s fakeSource) Points() ([]Point, error) { return s.points, nil }

func TestSourcesMergeInTimeOrder(t *testing.T) {
	photos := points(t, mine(t, "2025-03-01T10:00", bergen), mine(t, "2025-03-03T10:00", bergen))
	google := fakeSource{"google", []Point{{
		Source: "google", ID: "g", Time: time.Date(2025, 3, 2, 10, 0, 0, 0, time.UTC), LocalTime: "2025-03-02T11:00:00",
		Latitude: madrid.lat, Longitude: madrid.lon, City: "Madrid", Country: "Spain", AccuracyM: 20, Evidence: []string{"visit"},
	}}}
	merged, err := mergePoints([]Source{fakeSource{"immich", photos}, google})
	if err != nil {
		t.Fatal(err)
	}
	got := describe(Stays(merged, cityOptions()))
	if want := "Bergen, Norway 1 | Madrid, Spain 1 | Bergen, Norway 1"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestConfigNeedsASourceAndRefusesTheFlatFormat(t *testing.T) {
	for name, body := range map[string]string{
		"no source": `{}`,
		"flat":      `{"cameras": [{"make": "Apple"}], "people": ["Me"]}`,
		"empty":     `{"immich": {}}`,
		"no make":   `{"immich": {"cameras": [{"model": "X100V"}]}}`,
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, configFile), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadConfig(dir); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, configFile), []byte(`{"immich": {"people": ["Me"]}}`), 0o600)
	config, err := LoadConfig(dir)
	if err != nil || len(Sources(dir, config)) != 1 || Sources(dir, config)[0].Name() != "immich" {
		t.Errorf("got %+v, %v", config, err)
	}
}

func TestReadAPIKeyPrefersEnvironmentThenFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IMMICH_API_KEY", "")
	if _, err := readAPIKey(dir); err == nil {
		t.Error("expected an error with no key")
	}
	if err := os.WriteFile(filepath.Join(dir, "api-key"), []byte("from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if key, err := readAPIKey(dir); err != nil || key != "from-file" {
		t.Errorf("got %q, %v", key, err)
	}
	t.Setenv("IMMICH_API_KEY", "from-env")
	if key, err := readAPIKey(dir); err != nil || key != "from-env" {
		t.Errorf("got %q, %v", key, err)
	}
}
