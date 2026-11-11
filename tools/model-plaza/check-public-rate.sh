#!/bin/sh
# Exercise the production exchange-rate service against the real public API.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
probe=$(mktemp -d "$repo/backend/.model-plaza-rate-XXXXXX")
trap 'rm -rf "$probe"' EXIT HUP INT TERM
cat > "$probe/main.go" <<'GO'
package main
import (
 "context"
 "encoding/json"
 "fmt"
 "os"
 "time"
 "github.com/Wei-Shaw/sub2api/internal/service"
)
func main() {
 s := service.NewModelPlazaExchangeRateService()
 first := s.Get(context.Background())
 if first == nil || first.USDToCNY <= 0 || first.Source != "Frankfurter" { panic("public reference quote unavailable") }
 start := time.Now()
 cached := s.Get(context.Background())
 if cached == nil || *cached != *first || time.Since(start) > time.Second { panic("cache read failed") }
 receipt := struct { CheckedAt string `json:"checked_at"`; Quote *service.ModelPlazaExchangeRate `json:"quote"`; Cached bool `json:"cache_verified"`; Scope string `json:"scope"` }{time.Now().UTC().Format(time.RFC3339), first, true, "Production exchange-rate service with real Frankfurter API; no billing requests"}
 body, _ := json.MarshalIndent(receipt, "", "  ")
 if err := os.MkdirAll("/tmp/sub2api-model-plaza-e2e", 0700); err != nil { panic(err) }
 if err := os.WriteFile("/tmp/sub2api-model-plaza-e2e/public-rate.json", body, 0600); err != nil { panic(err) }
 fmt.Println(string(body))
}
GO
cd "$repo/backend"
go run "$probe/main.go"
