// Command plenka-server is a single-binary media server for a home router: torrent
// streaming, tracker search (Cardigann definitions downloaded at runtime), a TMDB
// proxy, shared watch history and the embedded web app.
package main

import (
	"context"
	"errors"
	"io/fs"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"plenka/internal/appconfig"
	"plenka/internal/buildinfo"
	"plenka/internal/config"
	"plenka/internal/discovery"
	"plenka/internal/history"
	"plenka/internal/httpserver"
	"plenka/internal/indexers"
	"plenka/internal/store"
	"plenka/internal/tmdb"
	"plenka/internal/torrents"
	tslog "plenka/internal/torrserver/log"
	tssettings "plenka/internal/torrserver/settings"
	"plenka/internal/torrserver/torr"
	"plenka/web"
)

func main() {
	if err := run(); err != nil {
		stdlog.Fatalf("plenka-server: %v", err)
	}
}

func run() error {
	cfg := config.Load()
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return err
	}
	tslog.Init(cfg.LogPath, "")
	tslog.TLogln("plenka-server", buildinfo.Version, "data dir", cfg.DataDir, "port", cfg.Port)

	// Torrent engine settings and its own database (config.db) live in the data dir.
	tssettings.Path = cfg.DataDir
	if err := tssettings.InitSets(); err != nil {
		return err
	}
	defer tssettings.CloseDB()

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()

	appCfg, err := appconfig.New(st, cfg.TMDBKey)
	if err != nil {
		return err
	}
	tmdbProxy := tmdb.New(appCfg.Get)
	hist := history.New(st)
	idx, err := indexers.New(indexers.Options{
		Store: st, Config: appCfg.Get, DataDir: cfg.DataDir, DefinitionsURL: cfg.DefinitionsURL,
	})
	if err != nil {
		return err
	}
	idx.StartDefinitionsUpdate() // tracker definitions are downloaded, not shipped
	torrents.IndexerLinkResolver = idx.ResolveLink

	bts := torr.NewBTS()
	if err := bts.Connect(); err != nil {
		return err
	}
	defer bts.Disconnect()

	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		return err
	}
	router := httpserver.New(httpserver.InfoSource{
		TMDBConfigured: tmdbProxy.Configured,
		IndexerStats:   idx.Stats,
	}, dist, appCfg, tmdbProxy, hist, idx, torrents.New())

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 15 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	// LAN discovery on the same port number over UDP. Not fatal: HTTP still works
	// when the port is taken, clients fall back to probing known addresses.
	if disc, err := discovery.Listen(":"+cfg.Port, buildinfo.Version, discovery.Port(cfg.Port)); err != nil {
		tslog.TLogln("discovery: udp listen failed:", err)
	} else {
		defer disc.Close()
		go func() {
			if err := disc.Serve(); err != nil {
				tslog.TLogln("discovery:", err)
			}
		}()
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case sig := <-stop:
		tslog.TLogln("shutting down on", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}
	return nil
}
