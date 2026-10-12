// Package migrations embeds the goose SQL migrations so the single binary
// can migrate its own database (`reputecast migrate up`).
package migrations

import "embed"

// FS holds every *.sql migration in this directory.
//
//go:embed *.sql
var FS embed.FS
