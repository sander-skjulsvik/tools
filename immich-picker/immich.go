package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/sander-skjulsvik/tools/libs/immich"
)

// allowedRequests is every request this tool can make to Immich. Search is the
// only POST, because Immich implements search as POST. The API key's permissions
// enforce the same limit on the server; this list makes it reviewable here.
var allowedRequests = []immich.Rule{
	immich.Allow(http.MethodGet, `^/api/search/person$`),
	immich.Allow(http.MethodGet, `^/api/assets/`+immich.UUIDPattern+`$`),
	immich.Allow(http.MethodGet, `^/api/assets/`+immich.UUIDPattern+`/thumbnail$`),
	immich.Allow(http.MethodPost, `^/api/search/metadata$`),
	immich.Allow(http.MethodPost, `^/api/search/smart$`),
}

// Client is a read-only Immich client limited to allowedRequests.
type Client struct {
	*immich.Client
}

// NewClient takes the server's root URL, without a path.
func NewClient(rawURL, apiKey string) (*Client, error) {
	client, err := immich.NewClient(rawURL, apiKey, allowedRequests)
	if err != nil {
		return nil, err
	}
	return &Client{client}, nil
}

// ResolvePerson accepts a person's name, ID, or Immich link, and returns the ID
// and a display name.
func (c *Client) ResolvePerson(person string) (id, name string, err error) {
	if found := regexp.MustCompile(immich.UUIDPattern).FindString(person); found != "" {
		return found, found, nil
	}
	data, _, err := c.Do(http.MethodGet, "/api/search/person", url.Values{"name": {person}}, nil)
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
		Items []json.RawMessage `json:"items"`
	} `json:"assets"`
}

func takenAfter(since time.Time) string {
	return since.Format("2006-01-02") + "T00:00:00.000Z"
}

// SearchPersonAssets returns every image of the person taken after since.
func (c *Client) SearchPersonAssets(personID string, since time.Time) ([]json.RawMessage, error) {
	var assets []json.RawMessage
	err := c.SearchMetadata(map[string]any{
		"personIds":  []string{personID},
		"type":       "IMAGE",
		"takenAfter": takenAfter(since),
		"withPeople": true,
		"withExif":   true,
		"order":      "asc",
	}, func(items []json.RawMessage) error {
		assets = append(assets, items...)
		return nil
	})
	return assets, err
}

// SearchQuery returns the IDs of the person's photos best matching a text query.
func (c *Client) SearchQuery(personID string, since time.Time, query string, limit int) ([]string, error) {
	data, _, err := c.Do(http.MethodPost, "/api/search/smart", nil, map[string]any{
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
	data, _, err := c.Do(http.MethodGet, "/api/assets/"+assetID, nil, nil)
	return data, err
}

// Preview returns an asset's preview image and content type.
func (c *Client) Preview(assetID string) ([]byte, string, error) {
	return c.Do(http.MethodGet, "/api/assets/"+assetID+"/thumbnail", url.Values{"size": {"preview"}}, nil)
}
