package service

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// ModelPlazaExchangeRate is a reference quote, never a billing exchange rate.
type ModelPlazaExchangeRate struct {
	USDToCNY float64 `json:"usd_to_cny"`
	Date     string  `json:"date"`
	Source   string  `json:"source"`
	Stale    bool    `json:"stale"`
}

// ModelPlazaExchangeRateService shares one bounded cache across plaza visitors.
type ModelPlazaExchangeRateService struct {
	mu          sync.Mutex
	client      *http.Client
	quote       *ModelPlazaExchangeRate
	nextAttempt time.Time
	lastSuccess time.Time
}

func NewModelPlazaExchangeRateService() *ModelPlazaExchangeRateService {
	return &ModelPlazaExchangeRateService{client: &http.Client{
		Timeout:       3 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func modelPlazaCNYPerCredit(raw string) float64 {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 || v > 1_000_000 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 1
	}
	return v
}

// Get refreshes every six hours and backs off failed requests for five minutes.
// Quotes older than seven days are unusable, including during upstream outages.
func (s *ModelPlazaExchangeRateService) Get(ctx context.Context) *ModelPlazaExchangeRate {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if !now.Before(s.nextAttempt) {
		s.nextAttempt = now.Add(5 * time.Minute)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.frankfurter.dev/v2/rate/USD/CNY", nil)
		if err == nil {
			res, fetchErr := s.client.Do(req)
			if fetchErr == nil {
				defer res.Body.Close()
				var body struct {
					Base  string  `json:"base"`
					Quote string  `json:"quote"`
					Date  string  `json:"date"`
					Rate  float64 `json:"rate"`
				}
				decodeErr := json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&body)
				date, dateErr := time.Parse("2006-01-02", body.Date)
				if res.StatusCode == http.StatusOK && decodeErr == nil && dateErr == nil &&
					body.Base == "USD" && body.Quote == "CNY" && body.Rate > 0 && body.Rate < 100 &&
					!date.After(now.Add(24*time.Hour)) && now.Sub(date) <= 7*24*time.Hour {
					s.quote = &ModelPlazaExchangeRate{USDToCNY: body.Rate, Date: body.Date, Source: "Frankfurter"}
					s.lastSuccess = now
					s.nextAttempt = now.Add(6 * time.Hour)
				}
			}
		}
	}
	if s.quote == nil {
		return nil
	}
	date, err := time.Parse("2006-01-02", s.quote.Date)
	if err != nil || now.Sub(date) > 7*24*time.Hour {
		return nil
	}
	quote := *s.quote
	quote.Stale = now.Sub(s.lastSuccess) >= 6*time.Hour
	return &quote
}
