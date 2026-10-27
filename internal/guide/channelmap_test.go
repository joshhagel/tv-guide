package guide

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/joshhagel/tv-guide/internal/hdhr"
)

func TestChannelMapLookup(t *testing.T) {
	m, err := NewChannelMap()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		guide    string
		callsign string
		wantOK   bool
	}{
		{"2.1", "KREM", true},
		{"2.9", "KREM", true},
		{"4.1", "KXLY", true},
		{"6.2", "KHQ", true},
		{"7.4", "KSPS", true},
		{"10.1", "KWSU", true},
		{"10.2", "", false},
		{"12.1", "KUID", true},
		{"14.3", "KQUP", true},
		{"22.7", "KSKN", true},
		{"26.1", "KCDT", true},
		{"28.2", "KAYU", true},
		{"32.4", "KDYS", true},
		{"34.8", "KGPX", true},
		{"29.1", "", false},
		{"99.1", "", false},
	}

	for _, tc := range tests {
		st, ok := m.Lookup(tc.guide)
		if ok != tc.wantOK {
			t.Fatalf("guide %q: got ok=%v want %v", tc.guide, ok, tc.wantOK)
		}
		if !tc.wantOK {
			continue
		}
		if st.Callsign != tc.callsign {
			t.Fatalf("guide %q: callsign %q want %q", tc.guide, st.Callsign, tc.callsign)
		}
	}
}

func TestBuildChannelMap(t *testing.T) {
	root := filepath.Join("..", "..", "fixtures")
	lineupBytes, err := os.ReadFile(filepath.Join(root, "lineup.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lineup hdhr.Lineup
	if err := json.Unmarshal(lineupBytes, &lineup); err != nil {
		t.Fatal(err)
	}
	xmltv, err := os.ReadFile(filepath.Join(root, "xmltv-sample.xml"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := BuildChannelMap(&lineup, xmltv)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"2.1":  "US35354.hdhomerun.com",
		"2.2":  "US70714.hdhomerun.com",
		"4.3":  "US62909.hdhomerun.com",
		"14.1": "US32020.hdhomerun.com",
		"32.1": "US17983.hdhomerun.com",
	}
	for gn, id := range want {
		if m[gn] != id {
			t.Fatalf("guide %s: got %q want %q", gn, m[gn], id)
		}
	}
}
