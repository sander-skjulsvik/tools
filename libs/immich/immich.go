// Package immich is a read-only client for the Immich API. Each tool passes the
// requests it may make; anything else is refused before it leaves the process.
package immich

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	// UUIDPattern matches an Immich ID, for use in Rule paths.
	UUIDPattern = `[0-9a-fA-F]{8}(?:-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}`
	// PageSize is the page size SearchMetadata asks for.
	PageSize    = 1000
	maxAttempts = 3
)

// Rule allows one method on paths matching a pattern.
type Rule struct {
	Method string
	Path   *regexp.Regexp
}

// Allow builds a Rule. The pattern should be anchored with ^ and $.
func Allow(method, pathPattern string) Rule {
	return Rule{Method: method, Path: regexp.MustCompile(pathPattern)}
}

// ErrForbiddenRequest means a request outside the client's rules was attempted.
var ErrForbiddenRequest = errors.New("request not allowed")

func allowed(rules []Rule, method, path string) bool {
	for _, rule := range rules {
		if method == rule.Method && rule.Path.MatchString(path) {
			return true
		}
	}
	return false
}

// guardTransport refuses requests outside the rules or to another host, before
// anything is sent. Every request the client makes passes through here.
type guardTransport struct {
	host  string
	rules []Rule
	next  http.RoundTripper
}

var _ http.RoundTripper = guardTransport{}

func (t guardTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host != t.host || !allowed(t.rules, r.Method, r.URL.Path) {
		return nil, fmt.Errorf("%w: %s %s%s", ErrForbiddenRequest, r.Method, r.URL.Host, r.URL.Path)
	}
	return t.next.RoundTrip(r)
}

// Client is an Immich client limited to its rules.
type Client struct {
	base   *url.URL
	apiKey string
	http   *http.Client
}

// NewClient takes the server's root URL, without a path, and the only requests
// the client may make.
func NewClient(rawURL, apiKey string, rules []Rule) (*Client, error) {
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
			Transport: guardTransport{host: base.Host, rules: rules, next: http.DefaultTransport},
			// Go would forward the x-api-key header on a redirect to another host,
			// so redirects are never followed. They surface as a status error.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

// BaseURL returns the server's root URL.
func (c *Client) BaseURL() string {
	return c.base.String()
}

// Do sends a request and returns the body and content type. It retries network
// errors and server errors. It never includes the API key in an error.
func (c *Client) Do(method, path string, query url.Values, body any) ([]byte, string, error) {
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

type searchResponse struct {
	Assets struct {
		Items    []json.RawMessage `json:"items"`
		NextPage *string           `json:"nextPage"`
	} `json:"assets"`
}

// SearchMetadata pages through POST /api/search/metadata, calling page with each
// page's assets. It sets size and page in a copy of body.
func (c *Client) SearchMetadata(body map[string]any, page func(items []json.RawMessage) error) error {
	request := maps.Clone(body)
	request["size"] = PageSize
	for number := 1; ; {
		request["page"] = number
		data, _, err := c.Do(http.MethodPost, "/api/search/metadata", nil, request)
		if err != nil {
			return err
		}
		var result searchResponse
		if err := json.Unmarshal(data, &result); err != nil {
			return fmt.Errorf("decoding search page %d: %w", number, err)
		}
		if err := page(result.Assets.Items); err != nil {
			return err
		}
		if result.Assets.NextPage == nil || *result.Assets.NextPage == "" {
			return nil
		}
		if number, err = strconv.Atoi(*result.Assets.NextPage); err != nil {
			return fmt.Errorf("unexpected next page %q", *result.Assets.NextPage)
		}
	}
}

// ReadAPIKey returns IMMICH_API_KEY, or else the file api-key in dataDir. The key
// lives next to the tool's data rather than in a config folder, because config
// folders are often kept in git.
func ReadAPIKey(dataDir string) (string, error) {
	if key := strings.TrimSpace(os.Getenv("IMMICH_API_KEY")); key != "" {
		return key, nil
	}
	path := filepath.Join(dataDir, "api-key")
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("no API key: set IMMICH_API_KEY or create %s", path)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "warning: other users can read %s; run chmod 600 on it\n", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("%s is empty", path)
	}
	return key, nil
}
