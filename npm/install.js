#!/usr/bin/env node
"use strict";

const fs = require("fs");
const path = require("path");

const DOWNLOAD_BASE = "__DOWNLOAD_BASE__";

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "x64",
  arm64: "arm64",
};

function getBinaryName() {
  const platform = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];

  if (!platform || !arch) {
    throw new Error(
      `Unsupported platform: ${process.platform} ${process.arch}`
    );
  }

  const ext = process.platform === "win32" ? ".exe" : "";
  return `cli-${platform}-${arch}${ext}`;
}

async function main() {
  try {
    const binaryName = getBinaryName();
    const url = `${DOWNLOAD_BASE}/${binaryName}`;
    const dest = path.join(
      __dirname,
      process.platform === "win32" ? "cli-binary.exe" : "cli-binary"
    );

    console.log(`Downloading ${binaryName}...`);

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 60000);

    const res = await fetch(url, {
      redirect: "follow",
      signal: controller.signal,
    });
    clearTimeout(timeout);

    if (!res.ok) {
      throw new Error(`Download failed with status ${res.status}`);
    }

    const buffer = Buffer.from(await res.arrayBuffer());
    fs.writeFileSync(dest, buffer);

    if (process.platform !== "win32") {
      fs.chmodSync(dest, 0o755);
    }

    console.log("Installation complete.");
  } catch (err) {
    console.warn(`Warning: failed to download binary: ${err.message}`);
    console.warn("You may need to install the binary manually.");
    process.exit(0);
  }
}

main();
