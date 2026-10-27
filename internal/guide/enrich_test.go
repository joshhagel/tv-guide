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
