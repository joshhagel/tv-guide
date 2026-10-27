package guide

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnrichXMLTVSpokaneOverlay(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "xmltv-sample.xml"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewChannelMap()
	if err != nil {
		t.Fatal(err)
	}
	out, err := EnrichXMLTV(raw, m, false)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "KREM 2 Spokane") {
		t.Fatalf("expected KREM overlay in output")
	}
	if !strings.Contains(s, "US35354.hdhomerun.com") {
		t.Fatalf("expected HDHomeRun channel id preserved")
	}
	// Remarshal must keep programme bodies — Jellyfin shows "undefined" without <title>.
	for _, title := range []string{"FBI", "CIA", "Star Trek", "Kenneth Copeland"} {
		if !strings.Contains(s, "<title>"+title+"</title>") {
			t.Fatalf("expected programme title %q preserved after enrich", title)
		}
	}
	if !strings.Contains(s, "<sub-title>Reticence</sub-title>") {
		t.Fatalf("expected programme sub-title preserved after enrich")
	}
}

func TestEnrichXMLTVFavoritesOnly(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "xmltv-sample.xml"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewChannelMap()
	if err != nil {
		t.Fatal(err)
	}
	out, err := EnrichXMLTV(raw, m, true)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "US90949.hdhomerun.com") {
		t.Fatalf("non-overlay channel 29.1 should be removed")
	}
	if !strings.Contains(s, "US35354.hdhomerun.com") {
		t.Fatalf("overlay channel 2.1 should remain")
	}
}
