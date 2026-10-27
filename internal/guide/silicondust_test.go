package guide

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSiliconDustClient_GunzipsResponse(t *testing.T) {
	plain := []byte(`<?xml version="1.0"?><tv></tv>`)
	var gzBuf bytes.Buffer
	zw := gzip.NewWriter(&gzBuf)
	if _, err := zw.Write(plain); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept-Encoding") != "gzip" {
			t.Errorf("Accept-Encoding=%q", r.Header.Get("Accept-Encoding"))
		}
		if r.URL.Query().Get("DeviceAuth") != "testauth" {
			t.Errorf("DeviceAuth=%q", r.URL.Query().Get("DeviceAuth"))
		}
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(gzBuf.Bytes())
	}))
	t.Cleanup(srv.Close)

	c := NewSiliconDustClient()
	c.baseURL = srv.URL
	c.http = srv.Client()

	got, err := c.FetchEPG(context.Background(), "testauth")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("got %q want %q", got, plain)
	}
}

func TestSiliconDustClient_ForbiddenIncludesHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)

	c := NewSiliconDustClient()
	c.baseURL = srv.URL
	c.http = srv.Client()

	_, err := c.FetchEPG(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("error=%v", err)
	}
	if !strings.Contains(err.Error(), "rate-limit") {
		t.Fatalf("expected rate-limit hint, got %v", err)
	}
}
