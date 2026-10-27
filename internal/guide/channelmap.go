package guide

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"

	"github.com/joshhagel/tv-guide/internal/data"
	"github.com/joshhagel/tv-guide/internal/hdhr"
)

var channelGuideNumRE = regexp.MustCompile(`^\d+\.\d+$`)

type ChannelMap struct {
	stations []data.OverlayStation
}

func NewChannelMap() (*ChannelMap, error) {
	stations, err := data.LoadOverlayStations()
	if err != nil {
		return nil, err
	}
	return &ChannelMap{stations: stations}, nil
}

func (m *ChannelMap) Stations() []data.OverlayStation {
	out := make([]data.OverlayStation, len(m.stations))
	copy(out, m.stations)
	return out
}

func (m *ChannelMap) Lookup(guideNumber string) (*data.OverlayStation, bool) {
	guideNumber = strings.TrimSpace(guideNumber)
	for i := range m.stations {
		s := &m.stations[i]
		if s.ExactOnly {
			if guideNumber == s.GuidePrefix+".1" {
				return s, true
			}
			continue
		}
		if guideNumber == s.GuidePrefix || strings.HasPrefix(guideNumber, s.GuidePrefix+".") {
			return s, true
		}
	}
	return nil, false
}

func (m *ChannelMap) IsOverlayChannel(guideNumber string) bool {
	_, ok := m.Lookup(guideNumber)
	return ok
}

type xmlTV struct {
	Channels []xmlChannel `xml:"channel"`
}

type xmlChannel struct {
	ID           string   `xml:"id,attr"`
	LCN          string   `xml:"lcn"`
	DisplayNames []string `xml:"display-name"`
}

func BuildChannelMap(lineup *hdhr.Lineup, xmltv []byte) (map[string]string, error) {
	if lineup == nil {
		return nil, fmt.Errorf("lineup required")
	}
	var doc xmlTV
	if err := xml.Unmarshal(xmltv, &doc); err != nil {
		return nil, err
	}
	byLCN := make(map[string]string)
	for _, ch := range doc.Channels {
		if ch.ID == "" {
			continue
		}
		lcn := strings.TrimSpace(ch.LCN)
		if lcn == "" {
			lcn = guideNumberFromDisplayNames(ch.DisplayNames)
		}
		if lcn != "" {
			byLCN[lcn] = ch.ID
		}
	}
	out := make(map[string]string)
	for _, lc := range lineup.Channels {
		gn := strings.TrimSpace(lc.GuideNumber)
		if id, ok := byLCN[gn]; ok {
			out[gn] = id
		}
	}
	return out, nil
}

func guideNumberFromDisplayNames(names []string) string {
	for _, n := range names {
		n = strings.TrimSpace(n)
		if channelGuideNumRE.MatchString(n) {
			return n
		}
		if i := strings.Index(n, " "); i > 0 {
			prefix := n[:i]
			if channelGuideNumRE.MatchString(prefix) {
				return prefix
			}
		}
	}
	return ""
}
