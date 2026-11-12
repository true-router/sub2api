// Acceptance uses real HTTP handlers, services, repositories and disposable PostgreSQL.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	listen := flag.String("listen", "", "Optional loopback address for browser acceptance")
	flag.Parse()
	dsn := os.Getenv("USAGE_SESSION_E2E_DSN")
	if !strings.Contains(dsn, "/usage_session_e2e?") {
		panic("requires disposable usage_session_e2e database")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	must(err)
	defer db.Close()
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	must(repository.ApplyMigrations(ctx, db))
	u1, err := client.User.Create().SetEmail("session-one@example.test").SetPasswordHash("fixture").Save(ctx)
	must(err)
	u2, err := client.User.Create().SetEmail("session-two@example.test").SetPasswordHash("fixture").Save(ctx)
	must(err)
	account, err := client.Account.Create().SetName("session-fixture").SetPlatform("anthropic").SetType("apikey").Save(ctx)
	must(err)
	key, err := client.APIKey.Create().SetUserID(u1.ID).SetKey("session-e2e-only").SetName("session fixture").Save(ctx)
	must(err)
	start := time.Now().UTC().Truncate(time.Minute).Add(-time.Hour)
	for i, sid := range []string{"session-shared", "session-other", "session-shared", "session-shared-prefix", "", "session-shared"} {
		uid := u1.ID
		if i == 5 {
			uid = u2.ID
		}
		b := client.UsageLog.Create().SetUserID(uid).SetAPIKeyID(key.ID).SetAccountID(account.ID).
			SetRequestID(fmt.Sprintf("request-%d", i)).SetModel("claude-sonnet-5-5").SetCreatedAt(start.Add(time.Duration(i) * time.Minute)).
			SetInputTokens(10).SetCacheReadTokens(100 * i).SetCacheCreationTokens(20).SetActualCost(0.01)
		row, err := b.Save(ctx)
		must(err)
		if sid != "" {
			_, err = db.ExecContext(ctx, "UPDATE usage_logs SET session_id=$1 WHERE id=$2", sid, row.ID)
			must(err)
		}
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	usage := service.NewUsageService(repository.NewUsageLogRepository(client, db), nil, client, nil)
	handler := adminhandler.NewUsageHandler(usage, nil, nil, nil)
	router.GET("/api/v1/admin/usage", handler.List)
	router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"ready": true}) })
	router.GET("/api/v1/admin/users/:id", func(c *gin.Context) {
		email := u1.Email
		if c.Param("id") == fmt.Sprint(u2.ID) {
			email = u2.Email
		}
		c.JSON(200, gin.H{"code": 0, "data": gin.H{"email": email}})
	})
	server := httptest.NewServer(router)
	defer server.Close()
	type result struct {
		Name   string `json:"name"`
		Passed bool   `json:"passed"`
	}
	results := []result{}
	check := func(name string, ok bool) {
		results = append(results, result{name, ok})
		if !ok {
			panic(name)
		}
		fmt.Println("PASS " + name)
	}
	defer func() {
		path := os.Getenv("USAGE_SESSION_E2E_RECEIPT")
		if path == "" {
			path = "/tmp/usage-session-e2e.json"
		}
		data, _ := json.MarshalIndent(map[string]any{"checks": results, "mechanism": "HTTP -> real handler -> real service -> real repository -> PostgreSQL", "production_modified": false}, "", "  ")
		must(os.WriteFile(path, data, 0600))
	}()
	type item struct {
		ID        int64   `json:"id"`
		UserID    int64   `json:"user_id"`
		SessionID *string `json:"session_id"`
		RequestID string  `json:"request_id"`
	}
	type page struct {
		Items []item `json:"items"`
		Total int    `json:"total"`
	}
	get := func(query string) page {
		resp, err := http.Get(server.URL + "/api/v1/admin/usage?" + query)
		must(err)
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			panic(fmt.Sprintf("HTTP %d", resp.StatusCode))
		}
		var body struct {
			Data page `json:"data"`
		}
		must(json.NewDecoder(resp.Body).Decode(&body))
		return body.Data
	}
	base := fmt.Sprintf("user_id=%d&session_id=session-shared&sort_by=created_at&sort_order=asc", u1.ID)
	p := get(base)
	check("exact session filter excludes other sessions, prefixes, NULLs and other users", p.Total == 2 && len(p.Items) == 2 && p.Items[0].RequestID == "request-0" && p.Items[1].RequestID == "request-2")
	check("session ID survives HTTP serialization", p.Items[0].SessionID != nil && *p.Items[0].SessionID == "session-shared")
	p = get(base + "&page=2&page_size=1&exact_total=true")
	check("filtered pagination and export totals remain exact", p.Total == 2 && len(p.Items) == 1 && p.Items[0].RequestID == "request-2")
	p = get("session_id=session-shared&sort_by=created_at&sort_order=asc")
	check("manual exact session query uses exact total without user filter", p.Total == 3 && len(p.Items) == 3)
	p = get("session_id=" + url.QueryEscape(" session-shared ") + fmt.Sprintf("&user_id=%d", u1.ID))
	check("whitespace around pasted identifiers is trimmed", p.Total == 2)
	p = get("session_id=" + url.QueryEscape("' OR 1=1 --"))
	check("session identifier is a bound SQL parameter", p.Total == 0 && len(p.Items) == 0)
	p = get(fmt.Sprintf("user_id=%d", u1.ID))
	check("clearing the session restores all user records", p.Total == 5 && len(p.Items) == 5)
	nulls := 0
	for _, v := range p.Items {
		if v.SessionID == nil {
			nulls++
		}
	}
	check("missing client session stays absent", nulls == 1)
	p = get(base + "&start_date=" + start.AddDate(0, 0, 1).Format("2006-01-02") + "&timezone=UTC")
	check("date bounds still apply to a session query", p.Total == 0)
	if *listen != "" {
		if !strings.HasPrefix(*listen, "127.0.0.1:") {
			panic("loopback only")
		}
		fmt.Println("READY http://" + *listen)
		must(http.ListenAndServe(*listen, router))
	}
}
