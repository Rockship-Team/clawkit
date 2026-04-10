// Package skills contains the skill files shipped with clawkit.
//
// The embedded FS is populated at build time from the subdirectories of
// this package (one directory per skill). It lets the installer copy
// skill files from the binary itself, so the globally-installed CLI
// works without network access or a local skills/ directory.
//
// When adding a new skill, add an entry to the //go:embed directive.
package skills

import "embed"

// FS holds the embedded skill directories. Use FS.ReadDir("skill-name")
// to enumerate a skill's files, and FS.ReadFile to read each one.
//
//go:embed all:finance-tracker all:gog all:shop-hoa all:carehub-baby
var FS embed.FS
