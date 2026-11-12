package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// registry lists the tags of one image through the OCI distribution API
// (/v2/<repo>/tags/list), with the token flow that GHCR and most registries use.
type registry struct {
	client *http.Client
	base   string // scheme and host, e.g. https://ghcr.io
	host   string // key in the auth file, e.g. ghcr.io
	repo   string // e.g. owner/image
	// authFile is a Docker config.json (the .dockerconfigjson of an image pull
	// secret). Read on every call, so a renewed secret is picked up without a
	// restart. Empty for public images.
	authFile string
}

// tags returns every tag of the image, following pagination.
func (r *registry) tags(ctx context.Context) ([]string, error) {
	next := r.base + "/v2/" + r.repo + "/tags/list?n=1000"
	var token string
	var all []string
	for next != "" {
		resp, err := r.get(ctx, next, token)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusUnauthorized && token == "" {
			challenge := resp.Header.Get("WWW-Authenticate")
			resp.Body.Close()
			if token, err = r.token(ctx, challenge); err != nil {
				return nil, err
			}
			continue // same page again, with the token
		}
		var page struct {
			Tags []string `json:"tags"`
		}
		err = decode(resp, &page)
		if err != nil {
			return nil, fmt.Errorf("listing tags: %w", err)
		}
		all = append(all, page.Tags...)
		next, err = r.nextPage(next, resp.Header.Get("Link"))
		if err != nil {
			return nil, err
		}
	}
	return all, nil
}

func (r *registry) get(ctx context.Context, u, token string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return r.client.Do(req)
}

// token fetches a pull token from the realm named in the registry's
// WWW-Authenticate challenge, with the credentials from the auth file if any.
func (r *registry) token(ctx context.Context, challenge string) (string, error) {
	scheme, params := parseChallenge(challenge)
	if !strings.EqualFold(scheme, "Bearer") || params["realm"] == "" {
		return "", fmt.Errorf("registry asked for unsupported auth %q", challenge)
	}
	u, err := url.Parse(params["realm"])
	if err != nil {
		return "", fmt.Errorf("token realm: %w", err)
	}
	q := u.Query()
	if s := params["service"]; s != "" {
		q.Set("service", s)
	}
	scope := params["scope"]
	if scope == "" {
		scope = "repository:" + r.repo + ":pull"
	}
	q.Set("scope", scope)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	basic, err := r.credentials()
	if err != nil {
		return "", err
	}
	if basic != "" {
		req.Header.Set("Authorization", "Basic "+basic)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	var body struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := decode(resp, &body); err != nil {
		return "", fmt.Errorf("getting a registry token (expired pull secret?): %w", err)
	}
	if body.Token != "" {
		return body.Token, nil
	}
	if body.AccessToken != "" {
		return body.AccessToken, nil
	}
	return "", errors.New("registry token response has no token")
}

// credentials returns base64(user:password) for r.host from the auth file,
// or "" when there is no auth file.
func (r *registry) credentials() (string, error) {
	if r.authFile == "" {
		return "", nil
	}
	data, err := os.ReadFile(r.authFile)
	if err != nil {
		return "", fmt.Errorf("reading auth file: %w", err)
	}
	var cfg struct {
		Auths map[string]struct {
			Auth     string `json:"auth"`
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"auths"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parsing auth file: %w", err)
	}
	for _, key := range []string{r.host, "https://" + r.host} {
		a, ok := cfg.Auths[key]
		if !ok {
			continue
		}
		if a.Auth != "" {
			return a.Auth, nil
		}
		if a.Username != "" {
			return base64.StdEncoding.EncodeToString([]byte(a.Username + ":" + a.Password)), nil
		}
	}
	return "", fmt.Errorf("auth file has no credentials for %s", r.host)
}

// nextPage resolves a Link: <…>; rel="next" header against the current page.
func (r *registry) nextPage(current, link string) (string, error) {
	if link == "" {
		return "", nil
	}
	target, rest, ok := strings.Cut(link, ";")
	if !ok || !strings.Contains(rest, `rel="next"`) {
		return "", nil
	}
	target = strings.Trim(strings.TrimSpace(target), "<>")
	base, err := url.Parse(current)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(target)
	if err != nil {
		return "", fmt.Errorf("pagination link: %w", err)
	}
	next := base.ResolveReference(ref)
	if next.Host != base.Host {
		return "", fmt.Errorf("pagination link points at another host: %s", next.Host)
	}
	return next.String(), nil
}

// parseChallenge splits `Bearer realm="…",service="…",scope="…"` into the
// scheme and its parameters.
func parseChallenge(h string) (string, map[string]string) {
	scheme, rest, _ := strings.Cut(strings.TrimSpace(h), " ")
	params := map[string]string{}
	for rest != "" {
		var key, value string
		key, rest, _ = strings.Cut(rest, "=")
		key = strings.ToLower(strings.TrimSpace(key))
		if strings.HasPrefix(rest, `"`) {
			value, rest, _ = strings.Cut(rest[1:], `"`)
			_, rest, _ = strings.Cut(rest, ",")
		} else {
			value, rest, _ = strings.Cut(rest, ",")
		}
		if key != "" {
			params[key] = strings.TrimSpace(value)
		}
	}
	return scheme, params
}

// decode reads a JSON body, turning non-2xx responses into errors.
func decode(resp *http.Response, v any) error {
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s: %s %s", resp.Request.Method, resp.Request.URL.Redacted(), resp.Status, strings.TrimSpace(string(msg)))
	}
	return json.NewDecoder(resp.Body).Decode(v)
}
