// Package skills contains the skill files shipped with clawkit.
//
// The embedded FS is populated at build time from the subdirectories of
// this package. Skills are grouped by vertical:
//
//	skills/ecommerce/shop-hoa/
//	skills/ecommerce/carehub-baby/
//	skills/utilities/finance-tracker/
//	skills/tools/gog/
//
// The installer looks up skills by name (not path) using findEmbeddedSkill.
//
// When adding a new skill, add its vertical to the //go:embed directive.
package skills

import (
	"embed"
	"io/fs"
)

//go:embed all:ecommerce all:finance all:real-estate all:sme all:study-aboard all:utilities
var FS embed.FS

// FindSkill searches the embedded FS for a skill by name across all verticals.
// Returns the FS path prefix (e.g. "ecommerce/shop-hoa") or empty string.
func FindSkill(name string) string {
	verticals, err := fs.ReadDir(FS, ".")
	if err != nil {
		return ""
	}
	for _, v := range verticals {
		if !v.IsDir() {
			continue
		}
		candidate := v.Name() + "/" + name
		if _, err := fs.Stat(FS, candidate); err == nil {
			return candidate
		}
	}
	return ""
}
