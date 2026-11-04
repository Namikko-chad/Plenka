// Package config reads the process configuration from environment variables.
package config

import (
	"os"
	"path/filepath"
)

// Config is the startup configuration.
type Config struct {
	// Port the HTTP server listens on (PORT, default 8095).
	Port string
	// DataDir holds plenka.db, the torrent engine's config.db and the disk cache (DATA_DIR, default ./data).
	DataDir string
	// TMDBKey overrides the TMDB key stored in settings when set (TMDB_KEY).
	TMDBKey string
	// LogPath writes the engine log to a file instead of stdout when set (LOG_PATH).
	LogPath string
	// DefinitionsURL is the .tar.gz the tracker definitions are downloaded from
	// (DEFINITIONS_URL, default: the Jackett repository snapshot on GitHub).
	DefinitionsURL string
}

// Load reads the environment.
func Load() Config {
	c := Config{
		Port:    env("PORT", "8095"),
		DataDir: env("DATA_DIR", "./data"),
		TMDBKey: os.Getenv("TMDB_KEY"),
		LogPath: os.Getenv("LOG_PATH"),

		DefinitionsURL: env("DEFINITIONS_URL", "https://codeload.github.com/Jackett/Jackett/tar.gz/refs/heads/master"),
	}
	if abs, err := filepath.Abs(c.DataDir); err == nil {
		c.DataDir = abs
	}
	return c
}

func env(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
