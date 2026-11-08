package immich

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestNewClientRejectsLinksWithPaths(t *testing.T) {
	if _, err := NewClient("https://photos.example.com/people/"+id, "key", rules); err == nil {
		t.Error("expected an error")
	}
	if _, err := NewClient("https://photos.example.com/", "key", rules); err != nil {
		t.Error(err)
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

func TestReadAPIKeyPrefersEnvironmentThenFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("IMMICH_API_KEY", "")
	if _, err := ReadAPIKey(dir); err == nil {
		t.Error("expected an error with no key")
	}
	if err := os.WriteFile(filepath.Join(dir, "api-key"), []byte("from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if key, err := ReadAPIKey(dir); err != nil || key != "from-file" {
		t.Errorf("got %q, %v", key, err)
	}
	t.Setenv("IMMICH_API_KEY", "from-env")
	if key, err := ReadAPIKey(dir); err != nil || key != "from-env" {
		t.Errorf("got %q, %v", key, err)
	}
}
