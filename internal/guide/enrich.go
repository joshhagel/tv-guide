package guide

import (
	"bytes"
	"encoding/xml"
	"regexp"
	"strings"

	"github.com/joshhagel/tv-guide/internal/data"
)

var hdhrID = regexp.MustCompile(`^US[0-9]+\.hdhomerun\.com$`)

type tvDoc struct {
	XMLName        xml.Name          `xml:"tv"`
	SourceInfoURL  string            `xml:"source-info-url,attr"`
	SourceInfoName string            `xml:"source-info-name,attr"`
	Channels       []enrichChannel   `xml:"channel"`
	Programmes     []enrichProgramme `xml:"programme"`
}

type enrichChannel struct {
	ID           string      `xml:"id,attr"`
	DisplayNames []string    `xml:"display-name"`
	LCN          string      `xml:"lcn"`
	URL          string      `xml:"url"`
	Icon         *enrichIcon `xml:"icon"`
}

type enrichIcon struct {
	Src string `xml:"src,attr"`
}

type enrichProgramme struct {
	Start   string `xml:"start,attr"`
	Stop    string `xml:"stop,attr"`
	Channel string `xml:"channel,attr"`
}

func EnrichChannel(ch enrichChannel, st data.OverlayStation, guideNumber string) enrichChannel {
	if !hdhrID.MatchString(ch.ID) {
		return ch
	}
	gn := strings.TrimSpace(guideNumber)
	if gn == "" {
		gn = strings.TrimSpace(ch.LCN)
	}
	if gn != "" {
		ch.LCN = gn
	}
	names := []string{}
	if gn != "" {
		names = append(names, gn)
	}
	if st.Name != "" {
		names = appendUniqueDisplay(names, st.Name)
	}
	if st.Callsign != "" {
		names = appendUniqueDisplay(names, st.Callsign)
	}
	if st.Network != "" {
		names = appendUniqueDisplay(names, st.Network)
	}
	for _, legacy := range ch.DisplayNames {
		names = appendUniqueDisplay(names, legacy)
	}
	ch.DisplayNames = names
	return ch
}

func appendUniqueDisplay(base []string, name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return base
	}
	for _, existing := range base {
		if strings.EqualFold(existing, name) {
			return base
		}
	}
	return append(base, name)
}

func EnrichXMLTV(raw []byte, cm *ChannelMap, favoritesOnly bool) ([]byte, error) {
	var doc tvDoc
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	var channels []enrichChannel
	keep := make(map[string]struct{})
	for _, ch := range doc.Channels {
		if !hdhrID.MatchString(ch.ID) {
			channels = append(channels, ch)
			keep[ch.ID] = struct{}{}
			continue
		}
		gn := strings.TrimSpace(ch.LCN)
		if gn == "" {
			gn = guideNumberFromDisplayNames(ch.DisplayNames)
		}
		st, ok := cm.Lookup(gn)
		if favoritesOnly && !ok {
			continue
		}
		if !ok {
			channels = append(channels, ch)
			keep[ch.ID] = struct{}{}
			continue
		}
		enriched := EnrichChannel(ch, *st, gn)
		channels = append(channels, enriched)
		keep[enriched.ID] = struct{}{}
	}
	doc.Channels = channels
	if favoritesOnly {
		var progs []enrichProgramme
		for _, p := range doc.Programmes {
			if _, ok := keep[p.Channel]; ok {
				progs = append(progs, p)
			}
		}
		doc.Programmes = progs
	}
	out, err := xml.MarshalIndent(doc, "", "\t")
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBufferString(xml.Header)
	buf.Write(out)
	return buf.Bytes(), nil
}
