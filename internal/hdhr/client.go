package hdhr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Discover struct {
	FriendlyName    string `json:"FriendlyName"`
	ModelNumber     string `json:"ModelNumber"`
	FirmwareName    string `json:"FirmwareName"`
	FirmwareVersion string `json:"FirmwareVersion"`
	DeviceID        string `json:"DeviceID"`
	DeviceAuth      string `json:"DeviceAuth"`
	BaseURL         string `json:"BaseURL"`
	LineupURL       string `json:"LineupURL"`
	TunerCount      int    `json:"TunerCount"`
}

type LineupChannel struct {
	GuideNumber string `json:"GuideNumber"`
	GuideName   string `json:"GuideName"`
	VideoCodec  string `json:"VideoCodec"`
	AudioCodec  string `json:"AudioCodec"`
	HD          int    `json:"HD"`
	Favorite    int    `json:"Favorite"`
	URL         string `json:"URL"`
}

type Lineup struct {
	Channels []LineupChannel `json:"Channels"`
}

func (l *Lineup) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '[' {
		return json.Unmarshal(data, &l.Channels)
	}
	type lineupAlias Lineup
	return json.Unmarshal(data, (*lineupAlias)(l))
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: timeout}}
}

func (c *Client) Discover(ctx context.Context) (*Discover, error) {
	var d Discover
	if err := c.getJSON(ctx, c.baseURL+"/discover.json", &d); err != nil {
		return nil, err
	}
	if d.DeviceAuth == "" {
		return nil, fmt.Errorf("discover: missing DeviceAuth")
	}
	return &d, nil
}

func (c *Client) Lineup(ctx context.Context) (*Lineup, error) {
	var l Lineup
	if err := c.getJSON(ctx, c.baseURL+"/lineup.json", &l); err != nil {
		return nil, err
	}
	return &l, nil
}

func (c *Client) getJSON(ctx context.Context, url string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("hdhr %s: %s: %s", url, resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}
