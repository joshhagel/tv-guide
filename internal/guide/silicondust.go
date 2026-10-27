package guide

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultSiliconDustXMLTV = "https://api.hdhomerun.com/api/xmltv"

// EPGFetcher downloads SiliconDust XMLTV for a DeviceAuth.
type EPGFetcher interface {
	FetchEPG(ctx context.Context, deviceAuth string) ([]byte, error)
}

// SiliconDustClient fetches guide data from api.hdhomerun.com.
type SiliconDustClient struct {
	http    *http.Client
	baseURL string
	email   string // optional Email+DeviceIDs path
}

// NewSiliconDustClient returns a client that advertises gzip (required by SiliconDust)
// and manually decompresses. DisableCompression keeps Accept-Encoding under our control.
func NewSiliconDustClient() *SiliconDustClient {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DisableCompression = true
	return &SiliconDustClient{
		http:    &http.Client{Timeout: 120 * time.Second, Transport: tr},
		baseURL: defaultSiliconDustXMLTV,
	}
}

// SetAccountAuth enables Email+DeviceIDs when a device ID is also provided at fetch time.
func (c *SiliconDustClient) SetAccountAuth(email string) {
	c.email = strings.TrimSpace(email)
}

// FetchEPG downloads XMLTV using DeviceAuth.
func (c *SiliconDustClient) FetchEPG(ctx context.Context, deviceAuth string) ([]byte, error) {
	return c.FetchEPGWithDevice(ctx, deviceAuth, "")
}

// FetchEPGWithDevice downloads XMLTV using DeviceAuth, or Email+DeviceIDs when email is configured.
func (c *SiliconDustClient) FetchEPGWithDevice(ctx context.Context, deviceAuth, deviceID string) ([]byte, error) {
	q := url.Values{}
	if c.email != "" && deviceID != "" {
		q.Set("Email", c.email)
		q.Set("DeviceIDs", deviceID)
	} else {
		if strings.TrimSpace(deviceAuth) == "" {
			return nil, fmt.Errorf("silicondust xmltv: empty DeviceAuth")
		}
		q.Set("DeviceAuth", deviceAuth)
	}

	endpoint := c.baseURL
	if endpoint == "" {
		endpoint = defaultSiliconDustXMLTV
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "tv-guide/1.0 (+https://github.com/joshhagel/tv-guide)")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		hint := ""
		if resp.StatusCode == http.StatusForbidden {
			hint = " (SiliconDust rate-limits frequent polls; wait a few minutes and avoid concurrent refreshes)"
		}
		return nil, fmt.Errorf("silicondust xmltv: HTTP %d: %s%s", resp.StatusCode, msg, hint)
	}

	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") || isGzip(body) {
		gz, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("silicondust xmltv: gzip: %w", err)
		}
		defer gz.Close()
		plain, err := io.ReadAll(gz)
		if err != nil {
			return nil, fmt.Errorf("silicondust xmltv: gunzip: %w", err)
		}
		return plain, nil
	}
	return body, nil
}

func isGzip(b []byte) bool {
	return len(b) >= 2 && b[0] == 0x1f && b[1] == 0x8b
}
