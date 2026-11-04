// Package definitions holds Plenka's own Cardigann tracker definitions. They are
// embedded in the binary and loaded on top of the downloaded set: an own definition
// wins over a downloaded one with the same id.
package definitions

import "embed"

//go:embed *.yml
var FS embed.FS
