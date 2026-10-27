package server

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"time"

	"github.com/joshhagel/tv-guide/internal/guide"
)

type Server struct {
	cache *guide.Cache
	now   func() time.Time
	mux   *http.ServeMux
}

func New(cache *guide.Cache) *Server {
	s := &Server{cache: cache, now: time.Now, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /status", s.handleStatus)
	s.mux.HandleFunc("GET /channels", s.handleChannels)
	s.mux.HandleFunc("GET /xmltv.xml", s.handleXMLTV)
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	st := s.cache.Status()
	out := map[string]any{
		"region":           "Spokane, WA",
		"updated_at":       st.UpdatedAt.UTC().Format(time.RFC3339),
		"last_attempt":     st.LastAttempt.UTC().Format(time.RFC3339),
		"stale":            st.Stale,
		"channel_count":    st.ChannelCount,
		"bytes":            st.Bytes,
		"last_error":       st.LastError,
	}
	if d := s.cache.Discover(); d != nil {
		out["device"] = map[string]string{
			"friendly_name": d.FriendlyName,
			"device_id":     d.DeviceID,
			"model":         d.ModelNumber,
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleChannels(w http.ResponseWriter, _ *http.Request) {
	views, err := s.cache.ChannelViews()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type row struct {
		GuideNumber string `json:"guide_number"`
		GuideName   string `json:"guide_name,omitempty"`
		XMLTVID     string `json:"xmltv_id,omitempty"`
	}
	rows := make([]row, 0, len(views))
	for _, v := range views {
		rows = append(rows, row{
			GuideNumber: v.GuideNumber,
			GuideName:   v.GuideName,
			XMLTVID:     v.XMLTVChannelID,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"channels": rows})
}

func (s *Server) handleXMLTV(w http.ResponseWriter, _ *http.Request) {
	// Never refresh SiliconDust from the request path — Jellyfin polls this URL and
	// concurrent/empty-cache refreshes cause HTTP 403 rate limits from api.hdhomerun.com.
	// Background refresh (with single-flight + backoff) owns upstream pulls.
	data, ok := s.cache.XMLTV()
	if !ok {
		st := s.cache.Status()
		msg := "guide not ready; background refresh in progress"
		if st.LastError != "" {
			msg = "guide not ready: " + st.LastError
		}
		w.Header().Set("Retry-After", "60")
		http.Error(w, msg, http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Encoding", "gzip")
	gw := gzip.NewWriter(w)
	defer gw.Close()
	_, _ = gw.Write(data)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
