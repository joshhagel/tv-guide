package guide

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/joshhagel/tv-guide/internal/hdhr"
)

type Status struct {
	UpdatedAt    time.Time `json:"updatedAt"`
	LastAttempt  time.Time `json:"lastAttempt"`
	LastError    string    `json:"lastError,omitempty"`
	Stale        bool      `json:"stale"`
	ChannelCount int       `json:"channelCount"`
	Bytes        int       `json:"bytes"`
}

type Cache struct {
	path          string
	refreshMin    time.Duration
	refreshMax    time.Duration
	favoritesOnly bool

	hdhr       *hdhr.Client
	epg        EPGFetcher
	channelMap *ChannelMap

	mu          sync.RWMutex
	data        []byte
	updatedAt   time.Time
	lastAttempt time.Time
	lastErr     error
	lineup      *hdhr.Lineup
	discover    *hdhr.Discover
	xmltvIDs    map[string]string

	// Single-flight refresh so concurrent callers cannot hammer SiliconDust.
	refreshMu sync.Mutex
	inflight  *refreshInflight
}

type refreshInflight struct {
	done chan struct{}
	err  error
}

func NewCache(
	path string,
	refreshMin, refreshMax time.Duration,
	favoritesOnly bool,
	hdhrClient *hdhr.Client,
	epg EPGFetcher,
	channelMap *ChannelMap,
) *Cache {
	return &Cache{
		path:          path,
		refreshMin:    refreshMin,
		refreshMax:    refreshMax,
		favoritesOnly: favoritesOnly,
		hdhr:          hdhrClient,
		epg:           epg,
		channelMap:    channelMap,
	}
}

func (c *Cache) LoadFromDisk() error {
	b, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	info, err := os.Stat(c.path)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = b
	c.updatedAt = info.ModTime()
	return nil
}

func (c *Cache) StartBackgroundRefresh(ctx context.Context) {
	go func() {
		_ = c.Refresh(ctx)
		for {
			wait := c.nextInterval()
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				_ = c.Refresh(ctx)
			}
		}
	}()
}

func (c *Cache) jitterInterval() time.Duration {
	min := c.refreshMin
	max := c.refreshMax
	if max <= min {
		return min
	}
	delta := max - min
	return min + time.Duration(rand.Int64N(int64(delta)+1))
}

// nextInterval chooses the delay until the next refresh. After a 403 from SiliconDust,
// use a shorter cooldown (15–30m) so recovery is possible without waiting a full day.
func (c *Cache) nextInterval() time.Duration {
	c.mu.RLock()
	err := c.lastErr
	c.mu.RUnlock()
	if err != nil && strings.Contains(err.Error(), "HTTP 403") {
		return 15*time.Minute + time.Duration(rand.Int64N(int64(15*time.Minute)+1))
	}
	return c.jitterInterval()
}

// Refresh runs at most one SiliconDust/HDHR pull at a time. Concurrent callers wait
// for the in-flight refresh result instead of issuing parallel API requests.
func (c *Cache) Refresh(ctx context.Context) error {
	c.refreshMu.Lock()
	if c.inflight != nil {
		inf := c.inflight
		c.refreshMu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-inf.done:
			return inf.err
		}
	}
	inf := &refreshInflight{done: make(chan struct{})}
	c.inflight = inf
	c.refreshMu.Unlock()

	inf.err = c.doRefresh(ctx)

	c.refreshMu.Lock()
	c.inflight = nil
	c.refreshMu.Unlock()
	close(inf.done)
	return inf.err
}

func (c *Cache) doRefresh(ctx context.Context) error {
	c.mu.Lock()
	c.lastAttempt = time.Now()
	c.mu.Unlock()

	disc, err := c.hdhr.Discover(ctx)
	if err != nil {
		c.recordFailure(err)
		return err
	}
	lineup, err := c.hdhr.Lineup(ctx)
	if err != nil {
		c.recordFailure(err)
		return err
	}

	var raw []byte
	if sd, ok := c.epg.(*SiliconDustClient); ok {
		raw, err = sd.FetchEPGWithDevice(ctx, disc.DeviceAuth, disc.DeviceID)
	} else {
		raw, err = c.epg.FetchEPG(ctx, disc.DeviceAuth)
	}
	if err != nil {
		c.recordFailure(err)
		return err
	}
	enriched, err := EnrichXMLTV(raw, c.channelMap, c.favoritesOnly)
	if err != nil {
		c.recordFailure(err)
		return err
	}
	idMap, err := BuildChannelMap(lineup, raw)
	if err != nil {
		c.recordFailure(err)
		return err
	}
	if err := c.writeAtomic(enriched); err != nil {
		c.recordFailure(err)
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = enriched
	c.updatedAt = time.Now()
	c.lastErr = nil
	c.lineup = lineup
	c.discover = disc
	c.xmltvIDs = idMap
	return nil
}

func (c *Cache) recordFailure(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastErr = err
}

func (c *Cache) writeAtomic(data []byte) error {
	dir := filepath.Dir(c.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

func (c *Cache) XMLTV() ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.data) == 0 {
		return nil, false
	}
	out := make([]byte, len(c.data))
	copy(out, c.data)
	return out, true
}

func (c *Cache) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	count := 0
	if c.lineup != nil {
		count = len(c.lineup.Channels)
	}
	st := Status{
		UpdatedAt:    c.updatedAt,
		LastAttempt:  c.lastAttempt,
		Stale:        c.lastErr != nil && len(c.data) > 0,
		ChannelCount: count,
		Bytes:        len(c.data),
	}
	if c.lastErr != nil {
		st.LastError = c.lastErr.Error()
	}
	return st
}

func (c *Cache) Lineup() *hdhr.Lineup {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lineup
}

func (c *Cache) Discover() *hdhr.Discover {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.discover == nil {
		return nil
	}
	d := *c.discover
	return &d
}

func (c *Cache) ChannelViews() ([]ChannelView, error) {
	if c.channelMap == nil {
		return nil, fmt.Errorf("channel map required")
	}
	c.mu.RLock()
	lineup := c.lineup
	idMap := c.xmltvIDs
	favOnly := c.favoritesOnly
	c.mu.RUnlock()
	if lineup == nil {
		return nil, nil
	}
	views := make([]ChannelView, 0, len(lineup.Channels))
	for _, ch := range lineup.Channels {
		if favOnly && ch.Favorite != 1 {
			continue
		}
		v := ChannelView{
			GuideNumber:    ch.GuideNumber,
			GuideName:      ch.GuideName,
			StreamURL:      ch.URL,
			HD:             ch.HD == 1,
			Favorite:       ch.Favorite == 1,
			XMLTVChannelID: idMap[ch.GuideNumber],
		}
		if st, ok := c.channelMap.Lookup(ch.GuideNumber); ok {
			v.OverlayCallsign = st.Callsign
			v.OverlayName = st.Name
			v.Network = st.Network
		}
		views = append(views, v)
	}
	return views, nil
}

type ChannelView struct {
	GuideNumber     string `json:"guideNumber"`
	GuideName       string `json:"guideName"`
	StreamURL       string `json:"streamUrl,omitempty"`
	HD              bool   `json:"hd"`
	Favorite        bool   `json:"favorite"`
	XMLTVChannelID  string `json:"xmltvChannelId,omitempty"`
	OverlayCallsign string `json:"overlayCallsign,omitempty"`
	OverlayName     string `json:"overlayName,omitempty"`
	Network         string `json:"network,omitempty"`
}
