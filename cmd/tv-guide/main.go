package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joshhagel/tv-guide/internal/config"
	"github.com/joshhagel/tv-guide/internal/guide"
	"github.com/joshhagel/tv-guide/internal/hdhr"
	"github.com/joshhagel/tv-guide/internal/server"
)

func main() {
	cfg := config.Load()

	channelMap, err := guide.NewChannelMap()
	if err != nil {
		log.Fatalf("channel map: %v", err)
	}

	hdhrClient := hdhr.NewClient(cfg.HDHRBaseURL, cfg.HTTPClientTimeout)
	epg := guide.NewSiliconDustClient()
	if email := os.Getenv("HDHR_ACCOUNT_EMAIL"); email != "" {
		epg.SetAccountAuth(email)
		log.Printf("using SiliconDust Email+DeviceIDs auth for %s", email)
	}
	cache := guide.NewCache(
		cfg.CachePath,
		cfg.RefreshMin,
		cfg.RefreshMax,
		cfg.FavoritesOnly,
		hdhrClient,
		epg,
		channelMap,
	)
	if err := cache.LoadFromDisk(); err != nil {
		log.Fatalf("load cache: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cache.StartBackgroundRefresh(ctx)

	srv := server.New(cache)
	httpServer := &http.Server{
		Addr:              cfg.ListenAddr(),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on %s (hdhr=%s cache=%s)", cfg.ListenAddr(), cfg.HDHRBaseURL, cfg.CachePath)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)
}
