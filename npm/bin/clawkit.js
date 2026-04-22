#!/usr/bin/env node
// Wrapper that executes the clawkit Go binary shipped with this npm package.
// It resolves the per-platform binary under ../binaries, points the binary
// at the skills/ and registry.json that ship alongside it, and forwards
// stdio + exit code.

const { spawnSync } = require("node:child_process");
const path = require("node:path");
const fs = require("node:fs");

function platformAsset() {
    const { platform, arch } = process;
    const os =
        platform === "darwin" ? "darwin" :
        platform === "linux"  ? "linux"  :
        platform === "win32"  ? "windows" : null;
    const a =
        arch === "x64"   ? "amd64" :
        arch === "arm64" ? "arm64" : null;
    if (!os || !a) {
        console.error(`clawkit: unsupported platform ${platform}/${arch}`);
        process.exit(1);
    }
    return `clawkit-${os}-${a}${os === "windows" ? ".exe" : ""}`;
}

const pkgRoot     = path.join(__dirname, "..");
const binariesDir = path.join(pkgRoot, "binaries");
const skillsDir   = path.join(pkgRoot, "skills");
const registry    = path.join(pkgRoot, "registry.json");
const binary      = path.join(binariesDir, platformAsset());

if (!fs.existsSync(binary)) {
    console.error(`clawkit: binary not found at ${binary}`);
    console.error("The npm package appears to be incomplete — try reinstalling:");
    console.error("  npm install -g @rockship-team/clawkit");
    process.exit(1);
}

if (process.platform !== "win32") {
    try { fs.chmodSync(binary, 0o755); } catch (_) { /* noop */ }
}

const env = { ...process.env };
if (fs.existsSync(skillsDir) && !env.CLAWKIT_SKILLS_DIR) {
    env.CLAWKIT_SKILLS_DIR = skillsDir;
}
if (fs.existsSync(registry) && !env.CLAWKIT_REGISTRY) {
    env.CLAWKIT_REGISTRY = registry;
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit", env });
if (result.error) {
    console.error(`clawkit: failed to execute binary — ${result.error.message}`);
    process.exit(1);
}
process.exit(result.status ?? 0);
