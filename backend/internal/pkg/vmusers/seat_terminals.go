package vmusers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

var seatID = regexp.MustCompile(`^[0-9a-f]{32}$`)
var slotID = regexp.MustCompile(`^[0-9a-f]{12}$`)

type Seat struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	Slot      string `json:"slot"`
	SessionID string `json:"session_id,omitempty"`
}

var terminalHTTP = &http.Client{
	Transport:     &http.Transport{Proxy: http.ProxyFromEnvironment, ResponseHeaderTimeout: 8 * time.Second},
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

func (c *Client) seatRequest(ctx context.Context, userID, id string, client *http.Client) (*http.Response, error) {
	path := "/api/panel"
	if userID != "" {
		path += "/users/" + url.PathEscape(userID)
	}
	path += "/seat-terminals"
	if id != "" {
		path += "/" + id
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Origin+path, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	return client.Do(request)
}

func (c *Client) TerminalSeats(ctx context.Context, userID string) ([]Seat, error) {
	upstream, err := c.seatRequest(ctx, userID, "", c.http)
	if err != nil {
		return nil, err
	}
	defer func() { _ = upstream.Body.Close() }()
	var body struct {
		Seats []Seat `json:"seats"`
	}
	if upstream.StatusCode != http.StatusOK {
		return nil, errTerminalUnavailable
	}
	if err := json.NewDecoder(io.LimitReader(upstream.Body, 256*1024)).Decode(&body); err != nil {
		return nil, err
	}
	seats := make([]Seat, 0, len(body.Seats))
	for _, item := range body.Seats {
		if seatID.MatchString(item.ID) && slotID.MatchString(item.Slot) && (item.State == "active" || item.State == "waiting_tools" || item.State == "idle") {
			seats = append(seats, item)
		}
	}
	return seats, nil
}

var errTerminalUnavailable = errors.New("terminal unavailable")

func (c *Client) OpenTerminal(ctx context.Context, userID, id string) (io.ReadCloser, int, int, int) {
	if !seatID.MatchString(id) {
		return nil, 0, 0, http.StatusNotFound
	}
	upstream, err := c.seatRequest(ctx, userID, id, terminalHTTP)
	if err != nil {
		return nil, 0, 0, http.StatusServiceUnavailable
	}
	columns, _ := strconv.Atoi(upstream.Header.Get("X-Terminal-Columns"))
	rows, _ := strconv.Atoi(upstream.Header.Get("X-Terminal-Rows"))
	if upstream.StatusCode == http.StatusOK && upstream.Header.Get("Content-Type") == "application/octet-stream" &&
		columns > 0 && columns <= 512 && rows > 0 && rows <= 256 {
		return upstream.Body, columns, rows, http.StatusOK
	}
	_ = upstream.Body.Close()
	if upstream.StatusCode == http.StatusNotFound || upstream.StatusCode == http.StatusTooManyRequests {
		return nil, 0, 0, upstream.StatusCode
	}
	return nil, 0, 0, http.StatusServiceUnavailable
}
