// This executable exercises real HTTP handlers, the update service, and the
// GitHub repository client against a local release server. It never installs a binary.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	admin "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type cache struct{ data string }

func (c *cache) GetUpdateInfo(context.Context) (string, error) { return c.data, nil }
func (c *cache) SetUpdateInfo(_ context.Context, data string, _ time.Duration) error {
	c.data = data
	return nil
}

type localTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t localTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Host != "api.github.com" {
		return nil, fmt.Errorf("unexpected remote host")
	}
	clone := request.Clone(request.Context())
	clone.URL.Scheme = t.target.Scheme
	clone.URL.Host = t.target.Host
	return t.base.RoundTrip(clone)
}

func main() {
	output := flag.String("output", "", "JSON evidence path")
	flag.Parse()
	if *output == "" {
		panic("--output required")
	}
	checks := []string{}
	check := func(name string, ok bool) {
		if !ok {
			panic(name)
		}
		checks = append(checks, name)
	}
	var mu sync.Mutex
	paths := []string{}
	latest := "tr-v0.2.26"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repos/true-router/sub2api/releases/latest":
			_ = json.NewEncoder(w).Encode(service.GitHubRelease{TagName: latest, HTMLURL: "https://github.com/true-router/sub2api/releases/tag/" + latest})
		case "/repos/true-router/sub2api/releases":
			_ = json.NewEncoder(w).Encode([]service.GitHubRelease{{TagName: "tr-v0.2.24"}, {TagName: "v0.2.23"}, {TagName: "unrelated-tag"}, {TagName: "tr-v0.2.22-rc1", Prerelease: true}})
		default:
			http.Error(w, "wrong repository", 404)
		}
	}))
	target, _ := url.Parse(upstream.URL)
	pooled, err := httpclient.GetClient(httpclient.Options{Timeout: 30 * time.Second})
	if err != nil {
		panic(err)
	}
	originalTransport := pooled.Transport
	pooled.Transport = localTransport{target: target, base: http.DefaultTransport}
	defer func() { pooled.Transport = originalTransport }()
	client := repository.NewGitHubReleaseClient("", false)
	oldCache, _ := json.Marshal(map[string]any{"latest": "99.0.0", "timestamp": time.Now().Unix(), "release_info": map[string]string{"html_url": "https://github.com/Wei-Shaw/sub2api/releases/tag/v99.0.0"}})
	store := &cache{data: string(oldCache)}
	updater := service.NewUpdateService(store, client, "0.2.25", "release")
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	handler := admin.NewSystemHandler(updater, nil)
	router.GET("/api/v1/admin/system/check-updates", handler.CheckUpdates)
	router.GET("/api/v1/admin/system/rollback-versions", handler.GetRollbackVersions)
	app := httptest.NewServer(router)
	get := func(path string) map[string]any {
		response, err := http.Get(app.URL + path)
		if err != nil {
			panic(err)
		}
		defer response.Body.Close()
		if response.StatusCode != 200 {
			panic(response.Status)
		}
		var body map[string]any
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			panic(err)
		}
		return body
	}
	info := get("/api/v1/admin/system/check-updates")["data"].(map[string]any)
	check("old upstream cache rejected", info["latest_version"] == "0.2.26" && info["has_update"] == true && info["cached"] == false)
	check("fork cache reusable", get("/api/v1/admin/system/check-updates")["data"].(map[string]any)["cached"] == true)
	versions := get("/api/v1/admin/system/rollback-versions")["data"].(map[string]any)["versions"].([]any)
	check("fork and historical tags normalize for rollback", len(versions) == 2 && versions[0].(map[string]any)["version"] == "0.2.24" && versions[1].(map[string]any)["version"] == "0.2.23")
	mu.Lock()
	latest = "v0.2.26"
	mu.Unlock()
	info = get("/api/v1/admin/system/check-updates?force=true")["data"].(map[string]any)
	check("historical fork latest remains readable", info["latest_version"] == "0.2.26")
	mu.Lock()
	latest = "upstream-unrelated"
	mu.Unlock()
	info = get("/api/v1/admin/system/check-updates?force=true")["data"].(map[string]any)
	check("invalid release tag never becomes update", info["latest_version"] == "0.2.26" && info["warning"] != "")
	mu.Lock()
	observed := append([]string(nil), paths...)
	mu.Unlock()
	check("all GitHub API requests target fork repository", len(observed) == 4)
	app.Close()
	upstream.Close()
	data, _ := json.MarshalIndent(map[string]any{"result": "PASS", "checks": checks, "github_paths": observed, "fixture_services_removed": true, "boundary": "Real HTTP handlers, update service and repository client; local GitHub fixture and memory cache; no installation or external network."}, "", "  ")
	if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*output, append(data, '\n'), 0644); err != nil {
		panic(err)
	}
	fmt.Println("PASS: update HTTP flow")
}
