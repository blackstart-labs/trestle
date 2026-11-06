// Package migrations embeds the forward-only SQL migrations.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
