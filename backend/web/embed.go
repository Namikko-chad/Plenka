// Package web embeds the built Vue app (frontend/dist, copied here by `make web`).
package web

import "embed"

// Dist holds the SPA files under "dist/".
//
//go:embed all:dist
var Dist embed.FS
