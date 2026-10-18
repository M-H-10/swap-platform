// Package migrations holds the versioned SQL files that build the database.
// The files are embedded into the Go program, so the compiled server always
// carries the exact migrations it was built with.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
