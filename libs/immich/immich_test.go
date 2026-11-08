package immich

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

const id = "11111111-1111-1111-1111-111111111111"

var rules = []Rule{
	Allow(http.MethodGet, `^/api/assets/`+UUIDPattern+`$`),
	Allow(http.MethodPost, `^/api/search/metadata$`),
}

func newClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, "secret-key", rules)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestUnlistedRequestsNeverLeaveTheClient(t *testing.T) {
	var calls []string
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) { calls = append(calls, r.URL.Path) })
	for _, request := range []struct{ method, path string }{
		{http.MethodPut, "/api/assets/" + id},
		{http.MethodDelete, "/api/assets"},
		{http.MethodPost, "/api/assets/" + id},
		{http.MethodPost, "/api/search/metadata/extra"},
		{http.MethodGet, "/api/assets/" + id + "/original"},
		{http.MethodGet, "/api/assets/" + id + "/../../tags"},
	} {
		if _, _, err := client.Do(request.method, request.path, nil, nil); !errors.Is(err, ErrForbiddenRequest) {
			t.Errorf("%s %s: got %v, want ErrForbiddenRequest", request.method, request.path, err)
		}
	}
	if len(calls) != 0 {
		t.Errorf("requests reached the server: %v", calls)
	}
}

func TestRedirectIsNotFollowedSoTheKeyStaysHome(t *testing.T) {
	reached := false
	elsewhere := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))
	defer elsewhere.Close()
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+r.URL.Path, http.StatusFound)
	})
	if _, _, err := client.Do(http.MethodGet, "/api/assets/"+id, nil, nil); err == nil {
		t.Error("expected an error for a redirect")
	}
	if reached {
		t.Error("the redirect was followed")
	}
}

func TestServerErrorsAreRetried(t *testing.T) {
	attempts := 0
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Write([]byte(`{"id":"x"}`))
	})
	if _, _, err := client.Do(http.MethodGet, "/api/assets/"+id, nil, nil); err != nil {
		t.Errorf("got %v after retry", err)
	}
	if attempts != 2 {
		t.Errorf("got %d attempts, want 2", attempts)
	}
}

func TestRejectedKeyExplainsItself(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
	_, _, err := client.Do(http.MethodGet, "/api/assets/"+id, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "permissions") {
		t.Errorf("got %v", err)
	}
	if strings.Contains(err.Error(), "secret-key") {
		t.Error("error contains the key")
	}
}

func TestNewClientAcceptsOnlyARootHTTPSURL(t *testing.T) {
	for _, good := range []string{
		"https://immich.example.org", "https://immich.example.org/", "https://192.168.1.20:2283",
		"https://photos.home", "http://localhost:2283", "http://127.0.0.1:2283", "http://[::1]:2283",
	} {
		if _, err := NewClient(good, "key", rules); err != nil {
			t.Errorf("%s: %v", good, err)
		}
	}
	for _, bad := range []string{
		"http://immich.example.org", "http://192.168.1.20:2283", "immich.example.org", "ftp://immich.example.org",
		"https://immich.example.org/people/" + id, "https://immich.example.org?x=1", "https://user:pw@immich.example.org", "https://",
	} {
		_, err := NewClient(bad, "key", rules)
		if err == nil {
			t.Errorf("%s: expected an error", bad)
		} else if strings.Contains(err.Error(), "photos") {
			t.Errorf("%s: the error suggests a host name: %v", bad, err)
		}
	}
	if _, err := NewClient("https://immich.example.org", " ", rules); err == nil {
		t.Error("an empty key was accepted")
	}
}

func TestSearchMetadataFollowsPagesAndKeepsTheBody(t *testing.T) {
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["withExif"] != true || body["size"] != float64(PageSize) {
			t.Errorf("unexpected body %v", body)
		}
		page := int(body["page"].(float64))
		var next any
		if page < 3 {
			next = strconv.Itoa(page + 1)
		}
		json.NewEncoder(w).Encode(map[string]any{"assets": map[string]any{
			"items": []map[string]int{{"page": page}}, "nextPage": next,
		}})
	})
	body := map[string]any{"withExif": true}
	var pages int
	err := client.SearchMetadata(body, func(items []json.RawMessage) error {
		pages += len(items)
		return nil
	})
	if err != nil || pages != 3 {
		t.Errorf("got %d pages, %v", pages, err)
	}
	if _, changed := body["page"]; changed {
		t.Error("the caller's body was changed")
	}
}

func TestSearchMetadataAlwaysEnds(t *testing.T) {
	for name, next := range map[string]func(page int) string{
		"same page":    func(page int) string { return strconv.Itoa(page) },
		"earlier page": func(page int) string { return "1" },
		"not a number": func(page int) string { return "next" },
		"endless":      func(page int) string { return strconv.Itoa(page + 1) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func(old int) { maxPages = old }(maxPages)
			maxPages = 5
			requests := 0
			client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				var body struct{ Page int }
				json.NewDecoder(r.Body).Decode(&body)
				json.NewEncoder(w).Encode(map[string]any{"assets": map[string]any{"items": []any{}, "nextPage": next(body.Page)}})
			})
			if err := client.SearchMetadata(map[string]any{}, func([]json.RawMessage) error { return nil }); err == nil {
				t.Error("expected an error")
			}
			if requests > maxPages {
				t.Errorf("made %d requests, more than %d pages", requests, maxPages)
			}
		})
	}
}
