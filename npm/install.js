#!/usr/bin/env node
"use strict";

const https = require("https");
const http = require("http");
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

function download(url) {
  return new Promise((resolve, reject) => {
    const client = url.startsWith("https") ? https : http;
    client
      .get(url, (res) => {
        // Follow redirects
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          return download(res.headers.location).then(resolve, reject);
        }

        if (res.statusCode !== 200) {
          reject(new Error(`Download failed with status ${res.statusCode}`));
          return;
        }

        const chunks = [];
        res.on("data", (chunk) => chunks.push(chunk));
        res.on("end", () => resolve(Buffer.concat(chunks)));
        res.on("error", reject);
      })
      .on("error", reject);
  });
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
    const data = await download(url);

    fs.writeFileSync(dest, data);

    if (process.platform !== "win32") {
      fs.chmodSync(dest, 0o755);
    }

    console.log("Installation complete.");
  } catch (err) {
    console.warn(`Warning: failed to download binary: ${err.message}`);
    console.warn("You may need to install the binary manually.");
    // Exit 0 so npm install doesn't fail
    process.exit(0);
  }
}

main();
