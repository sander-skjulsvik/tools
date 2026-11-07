package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	uuidPattern = `[0-9a-fA-F]{8}(?:-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}`
	pageSize    = 1000
	maxAttempts = 3
)

// allowedRequests is every request this tool can make to Immich. Search is the
// only POST, because Immich implements search as POST. The API key's permissions
// enforce the same limit on the server; this list makes it reviewable here.
var allowedRequests = []struct {
	method string
	path   *regexp.Regexp
}{
	{http.MethodGet, regexp.MustCompile(`^/api/search/person$`)},
	{http.MethodGet, regexp.MustCompile(`^/api/assets/` + uuidPattern + `$`)},
	{http.MethodGet, regexp.MustCompile(`^/api/assets/` + uuidPattern + `/thumbnail$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/search/metadata$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/search/smart$`)},
}

// ErrForbiddenRequest means a request outside allowedRequests was attempted.
var ErrForbiddenRequest = errors.New("request not allowed")

func requestAllowed(method, path string) bool {
	for _, rule := range allowedRequests {
		if method == rule.method && rule.path.MatchString(path) {
			return true
		}
	}
	return false
}

// guardTransport refuses requests outside allowedRequests or to another host,
// before anything is sent. Every request the client makes passes through here.
type guardTransport struct {
	host string
	next http.RoundTripper
}

func (t guardTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host != t.host || !requestAllowed(r.Method, r.URL.Path) {
		return nil, fmt.Errorf("%w: %s %s%s", ErrForbiddenRequest, r.Method, r.URL.Host, r.URL.Path)
	}
	return t.next.RoundTrip(r)
}

// Client is a read-only Immich client.
type Client struct {
	base   *url.URL
	apiKey string
	http   *http.Client
}

// NewClient takes the server's root URL, without a path.
func NewClient(rawURL, apiKey string) (*Client, error) {
	base, err := url.Parse(strings.TrimRight(rawURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parsing server URL: %w", err)
	}
	if (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" || base.Path != "" {
		return nil, fmt.Errorf("server URL must look like https://photos.example.com, got %q", rawURL)
	}
	return &Client{
		base:   base,
		apiKey: apiKey,
		http: &http.Client{
			Timeout:   2 * time.Minute,
			Transport: guardTransport{host: base.Host, next: http.DefaultTransport},
			// Go would forward the x-api-key header on a redirect to another host,
			// so redirects are never followed. They surface as a status error.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

// do sends a request and returns the body and content type. It retries network
// errors and server errors. It never includes the API key in an error.
func (c *Client) do(method, path string, query url.Values, body any) ([]byte, string, error) {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return nil, "", err
		}
	}
	target := *c.base
	target.Path = path
	target.RawQuery = query.Encode()

	var lastErr error
	for attempt := range maxAttempts {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		request, err := http.NewRequest(method, target.String(), bytes.NewReader(payload))
		if err != nil {
			return nil, "", err
		}
		request.Header.Set("x-api-key", c.apiKey)
		request.Header.Set("Accept", "application/json")
		if body != nil {
			request.Header.Set("Content-Type", "application/json")
		}

		response, err := c.http.Do(request)
		if err != nil {
			if errors.Is(err, ErrForbiddenRequest) {
				return nil, "", err
			}
			lastErr = fmt.Errorf("%s %s: %w", method, path, err)
			continue
		}
		data, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		switch {
		case response.StatusCode >= 500:
			lastErr = fmt.Errorf("%s %s: status %d", method, path, response.StatusCode)
		case response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden:
			return nil, "", fmt.Errorf("%s %s: status %d: check the API key and its permissions", method, path, response.StatusCode)
		case response.StatusCode >= 300:
			return nil, "", fmt.Errorf("%s %s: status %d: %s", method, path, response.StatusCode, snippet(data))
		case readErr != nil:
			lastErr = fmt.Errorf("%s %s: reading response: %w", method, path, readErr)
		default:
			return data, response.Header.Get("Content-Type"), nil
		}
	}
	return nil, "", lastErr
}

func snippet(data []byte) string {
	const limit = 300
	if len(data) > limit {
		data = data[:limit]
	}
	return strings.TrimSpace(string(data))
}

// ResolvePerson accepts a person's name, ID, or Immich link, and returns the ID
// and a display name.
func (c *Client) ResolvePerson(person string) (id, name string, err error) {
	if found := regexp.MustCompile(uuidPattern).FindString(person); found != "" {
		return found, found, nil
	}
	data, _, err := c.do(http.MethodGet, "/api/search/person", url.Values{"name": {person}}, nil)
	if err != nil {
		return "", "", err
	}
	var matches []struct{ ID, Name string }
	if err := json.Unmarshal(data, &matches); err != nil {
		return "", "", fmt.Errorf("decoding people: %w", err)
	}
	candidates := matches
	for _, match := range matches {
		if strings.EqualFold(match.Name, person) {
			candidates = []struct{ ID, Name string }{match}
			break
		}
	}
	if len(candidates) != 1 {
		var names []string
		for _, candidate := range candidates {
			names = append(names, fmt.Sprintf("%s (%s)", candidate.Name, candidate.ID))
		}
		return "", "", fmt.Errorf("expected one person named %q, found %d: %s; pass the ID instead",
			person, len(candidates), strings.Join(names, ", "))
	}
	return candidates[0].ID, candidates[0].Name, nil
}

type searchResponse struct {
	Assets struct {
		Items    []json.RawMessage `json:"items"`
		NextPage *string           `json:"nextPage"`
	} `json:"assets"`
}

func takenAfter(since time.Time) string {
	return since.Format("2006-01-02") + "T00:00:00.000Z"
}

// SearchPersonAssets returns every image of the person taken after since.
func (c *Client) SearchPersonAssets(personID string, since time.Time) ([]json.RawMessage, error) {
	var assets []json.RawMessage
	for page := 1; ; {
		data, _, err := c.do(http.MethodPost, "/api/search/metadata", nil, map[string]any{
			"personIds":  []string{personID},
			"type":       "IMAGE",
			"takenAfter": takenAfter(since),
			"withPeople": true,
			"withExif":   true,
			"order":      "asc",
			"size":       pageSize,
			"page":       page,
		})
		if err != nil {
			return nil, err
		}
		var result searchResponse
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("decoding search page %d: %w", page, err)
		}
		assets = append(assets, result.Assets.Items...)
		if result.Assets.NextPage == nil || *result.Assets.NextPage == "" {
			return assets, nil
		}
		if page, err = strconv.Atoi(*result.Assets.NextPage); err != nil {
			return nil, fmt.Errorf("unexpected next page %q", *result.Assets.NextPage)
		}
	}
}

// SearchQuery returns the IDs of the person's photos best matching a text query.
func (c *Client) SearchQuery(personID string, since time.Time, query string, limit int) ([]string, error) {
	data, _, err := c.do(http.MethodPost, "/api/search/smart", nil, map[string]any{
		"query":      query,
		"personIds":  []string{personID},
		"type":       "IMAGE",
		"takenAfter": takenAfter(since),
		"size":       limit,
		"page":       1,
	})
	if err != nil {
		return nil, err
	}
	var result searchResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decoding search for %q: %w", query, err)
	}
	ids := make([]string, 0, len(result.Assets.Items))
	for _, item := range result.Assets.Items {
		var asset struct{ ID string }
		if err := json.Unmarshal(item, &asset); err != nil {
			return nil, fmt.Errorf("decoding search for %q: %w", query, err)
		}
		ids = append(ids, asset.ID)
	}
	return ids, nil
}

// GetAsset returns one asset's full details, including its faces.
func (c *Client) GetAsset(assetID string) (json.RawMessage, error) {
	data, _, err := c.do(http.MethodGet, "/api/assets/"+assetID, nil, nil)
	return data, err
}

// Preview returns an asset's preview image and content type.
func (c *Client) Preview(assetID string) ([]byte, string, error) {
	return c.do(http.MethodGet, "/api/assets/"+assetID+"/thumbnail", url.Values{"size": {"preview"}}, nil)
}
