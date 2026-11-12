package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestParseVersion(t *testing.T) {
	good := map[string]version{
		"v0.1.0":    {0, 1, 0},
		"v1.20.300": {1, 20, 300},
		"v0.0.0":    {0, 0, 0},
	}
	for tag, want := range good {
		got, ok := parseVersion(tag)
		if !ok || got != want {
			t.Errorf("parseVersion(%q) = %v, %v; want %v, true", tag, got, ok, want)
		}
	}
	for _, tag := range []string{"", "latest", "1.2.3", "v1.2", "v1.2.3.4", "v1.2.3-rc.1", "v01.2.3", "v1.+2.3", "v1.-2.3", "sha-abc", "v1..3"} {
		if _, ok := parseVersion(tag); ok {
			t.Errorf("parseVersion(%q) accepted a tag that isn't a release", tag)
		}
	}
}

func TestNewestComparesNumerically(t *testing.T) {
	got, ok := newest([]string{"latest", "v0.9.0", "v0.10.0", "v0.2.11", "v1.0.0-rc.1"})
	if !ok || got.String() != "v0.10.0" {
		t.Fatalf("newest = %v, %v; want v0.10.0", got, ok)
	}
	if _, ok := newest([]string{"latest", "main"}); ok {
		t.Fatal("newest found a release among tags without one")
	}
}

func TestParseChallenge(t *testing.T) {
	scheme, p := parseChallenge(`Bearer realm="https://registry.example.com/token",service="registry.example.com",scope="repository:owner/app:pull"`)
	if scheme != "Bearer" || p["realm"] != "https://registry.example.com/token" ||
		p["service"] != "registry.example.com" || p["scope"] != "repository:owner/app:pull" {
		t.Fatalf("got %q %v", scheme, p)
	}
}

// fakes is a registry and a Kubernetes API in one test server.
type fakes struct {
	srv  *httptest.Server
	user string // credentials the token endpoint accepts
	pass string

	mu          sync.Mutex
	image       string
	annotations map[string]string
	patches     []string
	registryHit bool
}

func newFakes(t *testing.T, image string) *fakes {
	f := &fakes{user: "me", pass: "secret", image: image, annotations: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/owner/app/tags/list", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.registryHit = true
		f.mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s/token",service="reg",scope="repository:owner/app:pull"`, f.srv.URL))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Query().Get("last") == "" {
			w.Header().Set("Link", `</v2/owner/app/tags/list?last=v0.2.0&n=1000>; rel="next"`)
			json.NewEncoder(w).Encode(map[string]any{"tags": []string{"latest", "v0.1.0", "v0.10.0", "sha-abc"}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"tags": []string{"v0.9.0", "v1.0.0-rc.1"}})
	})
	mux.HandleFunc("GET /token", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		q := r.URL.Query()
		if !ok || user != f.user || pass != f.pass || q.Get("service") != "reg" || q.Get("scope") != "repository:owner/app:pull" {
			http.Error(w, "denied", http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
	})
	const deploymentPath = "/apis/apps/v1/namespaces/ns/deployments/app"
	mux.HandleFunc("GET "+deploymentPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sa-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		var d deployment
		d.Metadata.Annotations = f.annotations
		d.Spec.Template.Spec.Containers = []container{{Name: "sidecar", Image: "example.com/other:v9.9.9"}, {Name: "app", Image: f.image}}
		json.NewEncoder(w).Encode(d)
	})
	mux.HandleFunc("PATCH "+deploymentPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/strategic-merge-patch+json" {
			http.Error(w, "wrong content type", http.StatusUnsupportedMediaType)
			return
		}
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.patches = append(f.patches, string(body))
		f.mu.Unlock()
		w.Write([]byte("{}"))
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakes) deployer(t *testing.T) *deployer {
	dir := t.TempDir()
	token := filepath.Join(dir, "token")
	if err := os.WriteFile(token, []byte("sa-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(f.srv.URL)
	auth := filepath.Join(dir, "config.json")
	cfg := fmt.Sprintf(`{"auths":{%q:{"username":%q,"password":%q,"auth":%q}}}`,
		u.Host, f.user, f.pass, base64.StdEncoding.EncodeToString([]byte(f.user+":"+f.pass)))
	if err := os.WriteFile(auth, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	return &deployer{
		kube:       &kube{client: f.srv.Client(), base: f.srv.URL, tokenFile: token},
		registry:   &registry{client: f.srv.Client(), base: f.srv.URL, host: u.Host, repo: "owner/app", authFile: auth},
		image:      u.Host + "/owner/app",
		namespace:  "ns",
		deployment: "app",
		container:  "app",
	}
}

func imageOf(f *fakes, tag string) string {
	u, _ := url.Parse(f.srv.URL)
	return u.Host + "/owner/app:" + tag
}

func TestCheckRollsForwardToNewestRelease(t *testing.T) {
	f := newFakes(t, "")
	f.image = imageOf(f, "v0.2.0")
	if err := f.deployer(t).check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.patches) != 1 {
		t.Fatalf("got %d patches, want 1", len(f.patches))
	}
	var patch struct {
		Metadata struct {
			Annotations map[string]string `json:"annotations"`
		} `json:"metadata"`
		Spec struct {
			Template struct {
				Spec struct {
					Containers []container `json:"containers"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
	}
	if err := json.Unmarshal([]byte(f.patches[0]), &patch); err != nil {
		t.Fatal(err)
	}
	cs := patch.Spec.Template.Spec.Containers
	if len(cs) != 1 || cs[0].Name != "app" || cs[0].Image != imageOf(f, "v0.10.0") {
		t.Errorf("patched containers = %+v, want only app at v0.10.0", cs)
	}
	if got := patch.Metadata.Annotations["kubernetes.io/change-cause"]; got != "autodeploy: v0.10.0" {
		t.Errorf("change-cause = %q", got)
	}
}

func TestCheckLeavesCurrentOrNewerAlone(t *testing.T) {
	for _, tag := range []string{"v0.10.0", "v0.11.0"} {
		f := newFakes(t, "")
		f.image = imageOf(f, tag)
		if err := f.deployer(t).check(context.Background()); err != nil {
			t.Fatal(err)
		}
		if len(f.patches) != 0 {
			t.Errorf("running %s: patched %v, want no change", tag, f.patches)
		}
	}
}

func TestCheckReplacesNonReleaseTag(t *testing.T) {
	f := newFakes(t, "")
	f.image = imageOf(f, "latest")
	if err := f.deployer(t).check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.patches) != 1 || !strings.Contains(f.patches[0], "v0.10.0") {
		t.Fatalf("patches = %v, want one to v0.10.0", f.patches)
	}
}

func TestCheckPausedDoesNothing(t *testing.T) {
	f := newFakes(t, "")
	f.image = imageOf(f, "v0.1.0")
	f.annotations[pausedAnnotation] = "true"
	if err := f.deployer(t).check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.patches) != 0 || f.registryHit {
		t.Fatalf("paused, but patched %v / asked the registry %v", f.patches, f.registryHit)
	}
}

func TestCheckRefusesForeignImage(t *testing.T) {
	f := newFakes(t, "example.com/someone-else/app:v0.1.0")
	err := f.deployer(t).check(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not touching it") {
		t.Fatalf("err = %v, want a refusal", err)
	}
	if len(f.patches) != 0 {
		t.Fatalf("patched a container running another image: %v", f.patches)
	}
}

func TestCheckBadCredentials(t *testing.T) {
	f := newFakes(t, "")
	f.image = imageOf(f, "v0.1.0")
	d := f.deployer(t)
	f.pass = "renewed" // the token endpoint no longer accepts the file's password
	err := d.check(context.Background())
	if err == nil || !strings.Contains(err.Error(), "registry token") {
		t.Fatalf("err = %v, want a registry token error", err)
	}
	if len(f.patches) != 0 {
		t.Fatalf("patched without a tag list: %v", f.patches)
	}
}

func TestNextPageStaysOnHost(t *testing.T) {
	r := &registry{}
	if _, err := r.nextPage("https://registry.example.com/v2/a/tags/list", `<https://elsewhere.example.org/v2/a/tags/list?last=x>; rel="next"`); err == nil {
		t.Fatal("followed a pagination link to another host")
	}
	next, err := r.nextPage("https://registry.example.com/v2/a/tags/list?n=10", `</v2/a/tags/list?last=x&n=10>; rel="next"`)
	if err != nil || next != "https://registry.example.com/v2/a/tags/list?last=x&n=10" {
		t.Fatalf("next = %q, %v", next, err)
	}
}
