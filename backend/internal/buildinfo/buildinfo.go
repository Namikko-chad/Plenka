// Package buildinfo carries the server version, set at build time with
// -ldflags "-X plenka/internal/buildinfo.Version=<semver>".
package buildinfo

// Version of plenka-server.
var Version = "dev"
