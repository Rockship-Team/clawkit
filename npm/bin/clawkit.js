#!/usr/bin/env node
"use strict";

const path = require("path");
const { execFileSync } = require("child_process");
const fs = require("fs");
const os = require("os");

// Map Node.js platform/arch to binary name.
function getBinaryName() {
  const platform = os.platform(); // linux, darwin, win32
  const arch = os.arch();         // x64, arm64

  const platformMap = {
    linux: "linux",
    darwin: "darwin",
    win32: "windows",
  };

  const archMap = {
    x64: "amd64",
    arm64: "arm64",
  };

  const p = platformMap[platform];
  const a = archMap[arch];

  if (!p || !a) {
    console.error(`[clawkit] Unsupported platform: ${platform}/${arch}`);
    process.exit(1);
  }

  const name = `clawkit-${p}-${a}${platform === "win32" ? ".exe" : ""}`;
  return name;
}

const binaryName = getBinaryName();
const binaryPath = path.join(__dirname, "..", "binaries", binaryName);

if (!fs.existsSync(binaryPath)) {
  console.error(`[clawkit] Binary not found: ${binaryPath}`);
  console.error(`[clawkit] Please reinstall: npm install -g @rockship/clawkit`);
  process.exit(1);
}

// Ensure binary is executable (macOS/Linux).
if (process.platform !== "win32") {
  try {
    fs.chmodSync(binaryPath, 0o755);
  } catch (_) {}
}

try {
  execFileSync(binaryPath, process.argv.slice(2), { stdio: "inherit" });
} catch (err) {
  process.exit(err.status ?? 1);
}
