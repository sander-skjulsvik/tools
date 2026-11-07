package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	me    = "11111111-1111-1111-1111-111111111111"
	other = "22222222-2222-2222-2222-222222222222"
)

var since = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

func box(x1, y1, x2, y2 int) map[string]int {
	return map[string]int{
		"boundingBoxX1": x1, "boundingBoxY1": y1, "boundingBoxX2": x2, "boundingBoxY2": y2,
		"imageWidth": 1000, "imageHeight": 1000,
	}
}

func testID(n int) string {
	return fmt.Sprintf("aaaaaaaa-aaaa-aaaa-aaaa-%012d", n)
}

func asset(n int, faces []map[string]int, others []map[string]int) map[string]any {
	people := []map[string]any{{"id": me, "faces": faces}}
	if others != nil {
		people = append(people, map[string]any{"id": other, "faces": others})
	}
	return map[string]any{
		"id": testID(n), "checksum": "sum", "originalFileName": "photo.jpg",
		"originalPath": "/mnt/photos/photo.jpg", "localDateTime": "2025-01-01T12:00:00.000Z",
		"visibility": "timeline", "owner": map[string]string{"email": "dropped@example.com"},
		"people": people, "unassignedFaces": []any{},
	}
}

func goodFace() []map[string]int { return []map[string]int{box(300, 300, 600, 600)} }

func noiseJPEG(t *testing.T, seed int64) []byte {
	t.Helper()
	random := rand.New(rand.NewSource(seed))
	img := image.NewGray(image.Rect(0, 0, 1000, 1000))
	random.Read(img.Pix)
	return encodeJPEG(t, img)
}

func flatJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 1000, 1000))
	for i := range img.Pix {
		img.Pix[i] = 128
	}
	return encodeJPEG(t, img)
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, img, nil); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

// fakeImmich implements just enough of Immich, and counts what it receives.
type fakeImmich struct {
	t      *testing.T
	mutex  sync.Mutex
	assets []map[string]any
	images map[string][]byte
	calls  []string
}

func (f *fakeImmich) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.calls = append(f.calls, r.Method+" "+r.URL.Path)
	if r.Header.Get("x-api-key") != "key" {
		f.t.Errorf("missing API key on %s", r.URL.Path)
	}
	path := r.URL.Path
	switch {
	case path == "/api/search/person":
		json.NewEncoder(w).Encode([]map[string]string{{"id": me, "name": "Me"}, {"id": other, "name": "Me Too"}})
	case path == "/api/search/metadata":
		var body struct {
			Page       int
			PersonIDs  []string `json:"personIds"`
			WithPeople bool     `json:"withPeople"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if len(body.PersonIDs) != 1 || body.PersonIDs[0] != me || !body.WithPeople {
			f.t.Errorf("unexpected search body %+v", body)
		}
		const size = 2
		start := min((body.Page-1)*size, len(f.assets))
		end := min(start+size, len(f.assets))
		var next *string
		if end < len(f.assets) {
			value := strconv.Itoa(body.Page + 1)
			next = &value
		}
		json.NewEncoder(w).Encode(map[string]any{"assets": map[string]any{"items": f.assets[start:end], "nextPage": next}})
	case path == "/api/search/smart":
		json.NewEncoder(w).Encode(map[string]any{"assets": map[string]any{"items": []map[string]any{{"id": f.assets[0]["id"]}}}})
	case strings.HasSuffix(path, "/thumbnail"):
		w.Header().Set("Content-Type", "image/jpeg")
		data, ok := f.images[strings.Split(path, "/")[3]]
		if !ok {
			data = noiseJPEG(f.t, 1)
		}
		w.Write(data)
	default:
		id := strings.Split(path, "/")[3]
		for _, item := range f.assets {
			if item["id"] == id {
				json.NewEncoder(w).Encode(item)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeImmich) count(suffix string) int {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	total := 0
	for _, call := range f.calls {
		if strings.HasSuffix(call, suffix) {
			total++
		}
	}
	return total
}

func setup(t *testing.T, assets ...map[string]any) (*fakeImmich, *Client, *Catalog) {
	t.Helper()
	fake := &fakeImmich{t: t, assets: assets, images: map[string][]byte{}}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, "key")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := OpenCatalog(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}
	return fake, client, catalog
}

func collectAndAnalyze(t *testing.T, client *Client, catalog *Catalog, queries ...string) []*Record {
	t.Helper()
	options := CollectOptions{Person: me, Since: since, Queries: queries, QueryLimit: 10}
	if err := Collect(client, catalog, options); err != nil {
		t.Fatal(err)
	}
	if err := Analyze(client, catalog, 2); err != nil {
		t.Fatal(err)
	}
	records, err := catalog.RunRecords()
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func defaultOptions() ShortlistOptions {
	return ShortlistOptions{Limit: 150, MinFaceArea: 0.004, FaceAreaEnough: 0.02, MaxOtherFace: 0.25, DuplicateBits: 8}
}

func TestWritesAndUnlistedRequestsNeverLeaveTheClient(t *testing.T) {
	fake, client, _ := setup(t)
	for _, request := range []struct{ method, path string }{
		{http.MethodPut, "/api/assets/" + me},
		{http.MethodPut, "/api/assets"},
		{http.MethodDelete, "/api/assets"},
		{http.MethodPost, "/api/tags"},
		{http.MethodPut, "/api/tags/" + me + "/assets"},
		{http.MethodPost, "/api/search/metadata/extra"},
		{http.MethodPost, "/api/assets/" + me},
		{http.MethodGet, "/api/assets/" + me + "/original"},
		{http.MethodGet, "/api/assets/" + me + "/../../tags"},
	} {
		_, _, err := client.do(request.method, request.path, nil, nil)
		if !errors.Is(err, ErrForbiddenRequest) {
			t.Errorf("%s %s: got %v, want ErrForbiddenRequest", request.method, request.path, err)
		}
	}
	if len(fake.calls) != 0 {
		t.Errorf("requests reached the server: %v", fake.calls)
	}
}

func TestRedirectIsNotFollowedSoTheKeyStaysHome(t *testing.T) {
	reached := false
	elsewhere := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))
	defer elsewhere.Close()
	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+r.URL.Path, http.StatusFound)
	}))
	defer redirecting.Close()

	client, err := NewClient(redirecting.URL, "key")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetAsset(me); err == nil {
		t.Error("expected an error for a redirect")
	}
	if reached {
		t.Error("the redirect was followed")
	}
}

func TestServerErrorsAreRetried(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Write([]byte(`{"id":"x"}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "key")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetAsset(me); err != nil {
		t.Errorf("got %v after retry", err)
	}
	if attempts != 2 {
		t.Errorf("got %d attempts, want 2", attempts)
	}
}

func TestRejectedKeyExplainsItself(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "secret-key")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetAsset(me)
	if err == nil || !strings.Contains(err.Error(), "permissions") {
		t.Errorf("got %v", err)
	}
	if strings.Contains(err.Error(), "secret-key") {
		t.Error("error contains the key")
	}
}

func TestNewClientRejectsLinksWithPaths(t *testing.T) {
	if _, err := NewClient("https://photos.example.com/people/"+me, "key"); err == nil {
		t.Error("expected an error")
	}
	if _, err := NewClient("https://photos.example.com/", "key"); err != nil {
		t.Error(err)
	}
}

func TestResolvePerson(t *testing.T) {
	_, client, _ := setup(t)
	id, _, err := client.ResolvePerson("https://photos.example.com/people/" + other)
	if err != nil || id != other {
		t.Errorf("link: got %q, %v", id, err)
	}
	if id, _, err = client.ResolvePerson("me"); err != nil || id != me {
		t.Errorf("exact name: got %q, %v", id, err)
	}
	if _, _, err = client.ResolvePerson("Mx"); err == nil {
		t.Error("ambiguous name: expected an error")
	}
}

func TestCollectFollowsPagesAndRecordsQueries(t *testing.T) {
	_, client, catalog := setup(t, asset(0, goodFace(), nil), asset(1, goodFace(), nil), asset(2, goodFace(), nil),
		asset(3, goodFace(), nil), asset(4, goodFace(), nil))
	options := CollectOptions{Person: me, Since: since, Queries: []string{"hiking"}, QueryLimit: 10}
	if err := Collect(client, catalog, options); err != nil {
		t.Fatal(err)
	}
	run, err := catalog.LoadRun()
	if err != nil {
		t.Fatal(err)
	}
	if run.PersonID != me || len(run.AssetIDs) != 5 {
		t.Errorf("got %+v", run)
	}
	record, err := catalog.Load(testID(0))
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Queries) != 1 || record.Queries[0] != "hiking" {
		t.Errorf("queries: %v", record.Queries)
	}
	if strings.Contains(string(record.Immich), "dropped@example.com") {
		t.Error("owner details were kept")
	}
	if !strings.Contains(string(record.Immich), "originalPath") {
		t.Error("Immich data was lost")
	}
}

func TestSecondRunDoesNoRepeatWorkAndKeepsLabels(t *testing.T) {
	fake, client, catalog := setup(t, asset(0, goodFace(), nil))
	records := collectAndAnalyze(t, client, catalog)
	records[0].LLM = &Label{Keep: true}
	if err := catalog.Save(records[0]); err != nil {
		t.Fatal(err)
	}
	records = collectAndAnalyze(t, client, catalog)
	if got := fake.count("/thumbnail"); got != 1 {
		t.Errorf("downloaded the preview %d times", got)
	}
	if records[0].LLM == nil || !records[0].LLM.Keep {
		t.Error("label was lost")
	}
}

func TestChangedImageInvalidatesEarlierResults(t *testing.T) {
	item := asset(0, goodFace(), nil)
	fake, client, catalog := setup(t, item)
	records := collectAndAnalyze(t, client, catalog)
	records[0].LLM = &Label{Keep: true}
	if err := catalog.Save(records[0]); err != nil {
		t.Fatal(err)
	}
	item["checksum"] = "changed"
	records = collectAndAnalyze(t, client, catalog)
	if records[0].LLM != nil {
		t.Error("label survived a changed image")
	}
	if got := fake.count("/thumbnail"); got != 2 {
		t.Errorf("downloaded the preview %d times", got)
	}
}

func TestOutdatedMetricsReuseThePreview(t *testing.T) {
	fake, client, catalog := setup(t, asset(0, goodFace(), nil))
	records := collectAndAnalyze(t, client, catalog)
	records[0].Local.Version = metricsVersion - 1
	if err := catalog.Save(records[0]); err != nil {
		t.Fatal(err)
	}
	if err := Analyze(client, catalog, 1); err != nil {
		t.Fatal(err)
	}
	record, err := catalog.Load(testID(0))
	if err != nil {
		t.Fatal(err)
	}
	if record.Local.Version != metricsVersion {
		t.Error("metrics were not recomputed")
	}
	if got := fake.count("/thumbnail"); got != 1 {
		t.Errorf("downloaded the preview %d times", got)
	}
}

func TestMissingFaceBoxesAreFetchedFromTheAsset(t *testing.T) {
	summary := asset(0, goodFace(), nil)
	delete(summary, "people")
	full := asset(0, goodFace(), nil)
	fake, client, catalog := setup(t, summary)
	if err := Collect(client, catalog, CollectOptions{Person: me, Since: since}); err != nil {
		t.Fatal(err)
	}
	fake.assets = []map[string]any{full}
	if err := Analyze(client, catalog, 1); err != nil {
		t.Fatal(err)
	}
	record, err := catalog.Load(testID(0))
	if err != nil {
		t.Fatal(err)
	}
	if !record.Local.FaceFound {
		t.Error("face was not found")
	}
}

func decodeJPEG(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func parsedAsset(t *testing.T, item map[string]any) Asset {
	t.Helper()
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	var parsed Asset
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestSharpFaceScoresHigherThanFlatFace(t *testing.T) {
	item := parsedAsset(t, asset(0, goodFace(), nil))
	sharp := computeMetrics(decodeJPEG(t, noiseJPEG(t, 1)), item, me)
	flat := computeMetrics(decodeJPEG(t, flatJPEG(t)), item, me)
	if sharp.FaceSharpness <= flat.FaceSharpness*10 || sharp.FaceSharpness == 0 {
		t.Errorf("sharp %v, flat %v", sharp.FaceSharpness, flat.FaceSharpness)
	}
	if math.Abs(sharp.FaceArea-0.09) > 1e-9 {
		t.Errorf("face area %v", sharp.FaceArea)
	}
}

func TestOtherFacesAreMeasuredAgainstOwnFace(t *testing.T) {
	item := parsedAsset(t, asset(0, goodFace(), []map[string]int{box(0, 0, 150, 150)}))
	metrics := computeMetrics(decodeJPEG(t, noiseJPEG(t, 1)), item, me)
	if metrics.LargestOtherFaceRatio != 0.25 || metrics.OtherFaceCount != 1 {
		t.Errorf("got %+v", metrics)
	}
}

func TestSmallerOwnFaceCountsAsAnotherFace(t *testing.T) {
	item := parsedAsset(t, asset(0, []map[string]int{box(0, 0, 10, 10), box(300, 300, 600, 600)}, nil))
	own, others := splitFaces(item, me)
	if own == nil || own.BoundingBoxX1 != 300 || len(others) != 1 {
		t.Errorf("own %+v, others %d", own, len(others))
	}
}

func TestSimilarImagesHaveCloseHashes(t *testing.T) {
	first := toGray(decodeJPEG(t, noiseJPEG(t, 1)))
	second := toGray(decodeJPEG(t, noiseJPEG(t, 2)))
	if dhash(first) != dhash(toGray(decodeJPEG(t, noiseJPEG(t, 1)))) {
		t.Error("the same image hashed differently")
	}
	if dhash(first) == dhash(second) {
		t.Error("different images hashed the same")
	}
}

func TestCorruptPreviewFailsOnePhotoNotTheBatch(t *testing.T) {
	fake, client, catalog := setup(t, asset(0, goodFace(), nil), asset(1, goodFace(), nil))
	fake.images[testID(0)] = []byte("not an image")
	if err := Collect(client, catalog, CollectOptions{Person: me, Since: since}); err != nil {
		t.Fatal(err)
	}
	if err := Analyze(client, catalog, 1); err == nil {
		t.Error("expected an error reporting the failure")
	}
	record, err := catalog.Load(testID(1))
	if err != nil {
		t.Fatal(err)
	}
	if record.Local == nil {
		t.Error("the good photo was not analyzed")
	}
}

func TestShortlistFiltersGroupsAndTinyFaces(t *testing.T) {
	fake, client, catalog := setup(t,
		asset(0, goodFace(), nil),
		asset(1, goodFace(), []map[string]int{box(0, 0, 300, 300)}),
		asset(2, []map[string]int{box(0, 0, 50, 50)}, nil))
	for n := range 3 {
		fake.images[testID(n)] = noiseJPEG(t, int64(n))
	}
	entries, rejected, err := BuildShortlist(collectAndAnalyze(t, client, catalog), defaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].AssetID != testID(0) {
		t.Errorf("got %+v", entries)
	}
	if rejected[testID(1)] != "another prominent face" || rejected[testID(2)] != "face too small" {
		t.Errorf("got %v", rejected)
	}
}

func TestShortlistRemovesNearDuplicates(t *testing.T) {
	_, client, catalog := setup(t, asset(0, goodFace(), nil), asset(1, goodFace(), nil))
	entries, rejected, err := BuildShortlist(collectAndAnalyze(t, client, catalog), defaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || len(rejected) != 1 {
		t.Fatalf("entries %+v, rejected %v", entries, rejected)
	}
	for _, reason := range rejected {
		if !strings.HasPrefix(reason, nearDuplicatePrefix) {
			t.Errorf("reason %q", reason)
		}
	}
}

func TestShortlistTakesTurnsBetweenQueries(t *testing.T) {
	items := []map[string]any{}
	for n := range 6 {
		items = append(items, asset(n, goodFace(), nil))
	}
	fake, client, catalog := setup(t, items...)
	for n := range 6 {
		fake.images[testID(n)] = noiseJPEG(t, int64(n))
	}
	records := collectAndAnalyze(t, client, catalog)
	for index, record := range records {
		record.Queries = []string{"hiking"}
		if index == 5 {
			record.Queries = []string{"cooking"}
		}
	}
	options := defaultOptions()
	options.Limit = 2
	options.DuplicateBits = -1
	entries, rejected, err := BuildShortlist(records, options)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Queries[0]] = true
	}
	if !seen["hiking"] || !seen["cooking"] || len(entries) != 2 {
		t.Errorf("got %+v", entries)
	}
	if len(rejected) != 4 {
		t.Errorf("rejected %v", rejected)
	}
}

func TestShortlistKeepsOnePhotoPerImmichDuplicateGroup(t *testing.T) {
	first, second, third := asset(0, goodFace(), nil), asset(1, goodFace(), nil), asset(2, goodFace(), nil)
	first["duplicateId"], second["duplicateId"] = "group", "group"
	fake, client, catalog := setup(t, first, second, third)
	for n := range 3 {
		fake.images[testID(n)] = noiseJPEG(t, int64(n))
	}
	entries, rejected, err := BuildShortlist(collectAndAnalyze(t, client, catalog), defaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || len(rejected) != 1 {
		t.Fatalf("entries %+v, rejected %v", entries, rejected)
	}
	for _, reason := range rejected {
		if !strings.HasPrefix(reason, immichDuplicatePrefix) {
			t.Errorf("reason %q", reason)
		}
	}
}

func TestShortlistCapsPhotosPerOuting(t *testing.T) {
	times := []string{
		"2025-01-01T10:00:00.000Z", "2025-01-01T11:00:00.000Z", "2025-01-01T12:00:00.000Z",
		"2025-01-03T10:00:00.000Z",
	}
	items := []map[string]any{}
	for n, taken := range times {
		item := asset(n, goodFace(), nil)
		item["localDateTime"] = taken
		items = append(items, item)
	}
	fake, client, catalog := setup(t, items...)
	for n := range times {
		fake.images[testID(n)] = noiseJPEG(t, int64(n))
	}
	options := defaultOptions()
	options.MaxPerOuting = 2
	options.OutingGap = 6 * time.Hour
	entries, rejected, err := BuildShortlist(collectAndAnalyze(t, client, catalog), options)
	if err != nil {
		t.Fatal(err)
	}
	perOuting := map[int]int{}
	for _, entry := range entries {
		perOuting[entry.Outing]++
	}
	if len(entries) != 3 || len(perOuting) != 2 {
		t.Errorf("entries %+v", entries)
	}
	for outing, count := range perOuting {
		if count > 2 {
			t.Errorf("outing %d has %d photos", outing, count)
		}
	}
	if len(rejected) != 1 {
		t.Errorf("rejected %v", rejected)
	}
}

func TestShortlistKeepsOnePhotoPerBurstWithoutChaining(t *testing.T) {
	times := []string{
		"2025-01-01T10:00:00.000Z", "2025-01-01T10:01:30.000Z",
		"2025-01-01T10:03:00.000Z", "2025-01-01T10:04:30.000Z",
	}
	items := []map[string]any{}
	for n, taken := range times {
		item := asset(n, goodFace(), nil)
		item["localDateTime"] = taken
		items = append(items, item)
	}
	fake, client, catalog := setup(t, items...)
	for n := range times {
		fake.images[testID(n)] = noiseJPEG(t, int64(n))
	}
	options := defaultOptions()
	options.BurstWindow = 2 * time.Minute
	entries, rejected, err := BuildShortlist(collectAndAnalyze(t, client, catalog), options)
	if err != nil {
		t.Fatal(err)
	}
	// A long session must not collapse into a single "burst" by chaining.
	if len(entries) < 2 || len(entries) > 3 {
		t.Errorf("entries %+v", entries)
	}
	for _, reason := range rejected {
		if !strings.HasPrefix(reason, burstPrefix) {
			t.Errorf("reason %q", reason)
		}
	}
	taken := make([]time.Time, 0, len(entries))
	for _, entry := range entries {
		at, err := time.Parse(time.RFC3339, entry.Taken)
		if err != nil {
			t.Fatal(err)
		}
		for _, earlier := range taken {
			if at.Sub(earlier).Abs() <= options.BurstWindow {
				t.Errorf("kept photos %v and %v are in one burst", at, earlier)
			}
		}
		taken = append(taken, at)
	}
}

func TestShortlistRanksAndTakesTurnsWithinEachCamera(t *testing.T) {
	var items []map[string]any
	for n := range 8 {
		item := asset(n, goodFace(), nil)
		make := "phone"
		if n >= 4 {
			make = "camera"
		}
		item["exifInfo"] = map[string]string{"make": make}
		items = append(items, item)
	}
	fake, client, catalog := setup(t, items...)
	for n := range 8 {
		fake.images[testID(n)] = noiseJPEG(t, int64(n))
	}
	records := collectAndAnalyze(t, client, catalog)
	// Phones are far sharper than the camera, as selfies are in real data.
	for index, record := range records {
		record.Local.FaceSharpness = 100
		if index < 4 {
			record.Local.FaceSharpness = 1000 + float64(index)
		} else {
			record.Local.FaceSharpness = float64(index)
		}
	}
	options := defaultOptions()
	options.Limit = 4
	options.DuplicateBits = -1
	entries, _, err := BuildShortlist(records, options)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, entry := range entries {
		counts[entry.Camera]++
	}
	if counts["phone"] != 2 || counts["camera"] != 2 {
		t.Errorf("got %v", counts)
	}
	if entries[0].Camera == entries[1].Camera {
		t.Errorf("cameras did not take turns: %+v", entries[:2])
	}
}

func TestBigFacesEarnNoExtraCredit(t *testing.T) {
	big := asset(0, []map[string]int{box(0, 0, 800, 800)}, nil)
	enough := asset(1, []map[string]int{box(0, 0, 200, 200)}, nil)
	fake, client, catalog := setup(t, big, enough)
	for n := range 2 {
		fake.images[testID(n)] = noiseJPEG(t, int64(n))
	}
	records := collectAndAnalyze(t, client, catalog)
	for _, record := range records {
		record.Local.FaceSharpness = 100
	}
	options := defaultOptions()
	options.DuplicateBits = -1
	entries, _, err := BuildShortlist(records, options)
	if err != nil {
		t.Fatal(err)
	}
	// Both earn full size credit (0.3), and the sharpness tie-break gives 0 and 0.7.
	if entries[0].Score != 1 || entries[1].Score != 0.3 {
		t.Errorf("got %+v", entries)
	}
}

func TestLonePhotoFromACameraIsNotDroppedAsBlurriest(t *testing.T) {
	item := asset(0, goodFace(), nil)
	item["exifInfo"] = map[string]string{"make": "rare"}
	_, client, catalog := setup(t, item)
	options := defaultOptions()
	options.DropBlurriestPercent = 25
	entries, rejected, err := BuildShortlist(collectAndAnalyze(t, client, catalog), options)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("rejected %v", rejected)
	}
}

func TestSmallSharpFaceIsNotBlurredByUpscaling(t *testing.T) {
	small := parsedAsset(t, asset(0, []map[string]int{box(100, 100, 150, 150)}, nil))
	metrics := computeMetrics(decodeJPEG(t, noiseJPEG(t, 1)), small, me)
	large := parsedAsset(t, asset(0, []map[string]int{box(100, 100, 700, 700)}, nil))
	largeMetrics := computeMetrics(decodeJPEG(t, flatJPEG(t)), large, me)
	if metrics.FaceSharpness <= largeMetrics.FaceSharpness {
		t.Errorf("small sharp %v, large flat %v", metrics.FaceSharpness, largeMetrics.FaceSharpness)
	}
	if metrics.FaceSharpness < 100 {
		t.Errorf("small sharp face scored only %v", metrics.FaceSharpness)
	}
}

func TestPhotosLabelledExcludeAreNeverShortlisted(t *testing.T) {
	fake, client, catalog := setup(t, asset(0, goodFace(), nil), asset(1, goodFace(), nil))
	for n := range 2 {
		fake.images[testID(n)] = noiseJPEG(t, int64(n))
	}
	records := collectAndAnalyze(t, client, catalog)
	records[0].LLM = &Label{Category: excludeCategory}
	entries, rejected, err := BuildShortlist(records, defaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].AssetID != records[1].AssetID {
		t.Errorf("entries %+v", entries)
	}
	if rejected[records[0].AssetID] != "excluded by its label" {
		t.Errorf("rejected %v", rejected)
	}
}

func TestDeletedPreviewOfExcludedPhotoIsNotDownloadedAgain(t *testing.T) {
	fake, client, catalog := setup(t, asset(0, goodFace(), nil))
	records := collectAndAnalyze(t, client, catalog)
	records[0].LLM = &Label{Category: excludeCategory}
	if err := catalog.Save(records[0]); err != nil {
		t.Fatal(err)
	}
	path := catalog.PreviewPath(records[0])
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	// A new collect must keep the label, and analyze must not fetch the preview.
	if err := Collect(client, catalog, CollectOptions{Person: me, Since: since}); err != nil {
		t.Fatal(err)
	}
	if err := Analyze(client, catalog, 1); err != nil {
		t.Fatal(err)
	}
	if got := fake.count("/thumbnail"); got != 1 {
		t.Errorf("downloaded the preview %d times", got)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the preview came back: %v", err)
	}
	record, err := catalog.Load(testID(0))
	if err != nil {
		t.Fatal(err)
	}
	if record.LLM == nil || record.LLM.Category != excludeCategory {
		t.Errorf("the exclude label was lost: %+v", record.LLM)
	}
}

func TestOutingNumbersGiveUndatedPhotosTheirOwnOuting(t *testing.T) {
	numbers := outingNumbers([]Asset{
		{ID: "a", LocalDateTime: "2025-01-01T10:00:00.000Z"},
		{ID: "b", LocalDateTime: "2025-01-01T11:00:00.000Z"},
		{ID: "c"},
		{ID: "d"},
	}, 6*time.Hour)
	if numbers["a"] != numbers["b"] {
		t.Errorf("close photos were split: %v", numbers)
	}
	if numbers["c"] == numbers["d"] || numbers["c"] == numbers["a"] {
		t.Errorf("undated photos were grouped: %v", numbers)
	}
}

func TestDropBlurriestRemovesLowestSharpness(t *testing.T) {
	fake, client, catalog := setup(t, asset(0, goodFace(), nil), asset(1, goodFace(), nil))
	fake.images[testID(0)] = noiseJPEG(t, 1)
	fake.images[testID(1)] = flatJPEG(t)
	options := defaultOptions()
	options.DropBlurriestPercent = 25
	entries, rejected, err := BuildShortlist(collectAndAnalyze(t, client, catalog), options)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].AssetID != testID(0) || rejected[testID(1)] != "blurriest faces" {
		t.Errorf("entries %+v, rejected %v", entries, rejected)
	}
}

func TestWriteShortlistAndSheets(t *testing.T) {
	items := []map[string]any{}
	for n := range 10 {
		items = append(items, asset(n, goodFace(), nil))
	}
	fake, client, catalog := setup(t, items...)
	for n := range 10 {
		fake.images[testID(n)] = noiseJPEG(t, int64(n))
	}
	collectAndAnalyze(t, client, catalog)
	if err := WriteShortlist(catalog, defaultOptions()); err != nil {
		t.Fatal(err)
	}
	copies, err := os.ReadDir(catalog.shortlist)
	if err != nil || len(copies) != 10 {
		t.Fatalf("got %d shortlist files, %v", len(copies), err)
	}
	if !strings.HasPrefix(copies[0].Name(), "001_") {
		t.Errorf("first file %q", copies[0].Name())
	}

	if err := WriteSheets(catalog, 9); err != nil {
		t.Fatal(err)
	}
	sheets, err := os.ReadDir(catalog.sheets)
	if err != nil || len(sheets) != 2 {
		t.Fatalf("got %d sheets, %v", len(sheets), err)
	}
	data, err := os.ReadFile(filepath.Join(catalog.sheets, "sheet-001.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	sheet := decodeJPEG(t, data)
	if sheet.Bounds().Dx() != 3*sheetCellPixels || sheet.Bounds().Dy() != 3*sheetCellPixels {
		t.Errorf("sheet size %v", sheet.Bounds())
	}
	// The number's box is in the first cell's corner.
	if r, g, b, _ := sheet.At(0, 0).RGBA(); color.GrayModel.Convert(color.RGBA64{uint16(r), uint16(g), uint16(b), 0xffff}).(color.Gray).Y > 40 {
		t.Error("expected a dark number box in the corner")
	}
}

func goodLabel(id string) map[string]any {
	return map[string]any{
		"asset_id": id, "model": "m", "prompt_version": "1", "keep": true,
		"category": "portrait", "description": "Outdoors", "issues": []string{}, "edit_needed": false,
	}
}

func writeLabels(t *testing.T, labels ...map[string]any) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "labels.json")
	data, err := json.Marshal(labels)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImportLabelsRejectsWholeFileOnAnyError(t *testing.T) {
	_, client, catalog := setup(t, asset(0, goodFace(), nil))
	collectAndAnalyze(t, client, catalog)

	missing := goodLabel(testID(0))
	delete(missing, "keep")
	extra := goodLabel(testID(0))
	extra["surprise"] = true
	wrongType := goodLabel(testID(0))
	wrongType["keep"] = "yes"
	editWithoutNotes := goodLabel(testID(0))
	editWithoutNotes["edit_needed"] = true
	for name, bad := range map[string]map[string]any{
		"unknown asset":      goodLabel("nope"),
		"missing field":      missing,
		"unknown field":      extra,
		"wrong type":         wrongType,
		"edit without notes": editWithoutNotes,
	} {
		if err := ImportLabels(catalog, writeLabels(t, goodLabel(testID(0)), bad)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
		record, err := catalog.Load(testID(0))
		if err != nil {
			t.Fatal(err)
		}
		if record.LLM != nil {
			t.Errorf("%s: the good label was imported anyway", name)
		}
	}

	edited := goodLabel(testID(0))
	edited["edit_needed"] = true
	edited["edit_notes"] = "lift the shadows from the RAW"
	if err := ImportLabels(catalog, writeLabels(t, edited)); err != nil {
		t.Fatal(err)
	}
	record, err := catalog.Load(testID(0))
	if err != nil {
		t.Fatal(err)
	}
	if record.LLM == nil || record.LLM.Category != "portrait" || record.LLM.LabelledAt == "" {
		t.Errorf("got %+v", record.LLM)
	}
	if !record.LLM.EditNeeded || record.LLM.EditNotes == "" {
		t.Errorf("editing note was lost: %+v", record.LLM)
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
