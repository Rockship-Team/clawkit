// Package version exposes the clawkit build version. The value of Version is
// injected at build time via -ldflags "-X .../internal/version.Version=x.y.z"
// (see Makefile). The default "dev" is used for `go build` / `go install`
// without ldflags, during which case the installer falls back to "latest"
// GitHub release assets.
package version

// Version is the semver string of this binary, without a leading "v".
// Set via -ldflags at build time.
var Version = "dev"

// Tag returns the git tag form of Version ("vX.Y.Z"). If Version is "dev"
// or otherwise not a semver, Tag returns "latest" so URL resolution falls
// back to the latest release.
func Tag() string {
	if Version == "" || Version == "dev" {
		return "latest"
	}
	return "v" + Version
}

// IsDev reports whether this is a development build (no ldflags version).
func IsDev() bool {
	return Version == "" || Version == "dev"
}
