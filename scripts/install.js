#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");
const os = require("os");

const VERSION = require("../package.json").version;
const REPO = "tangcent/apilot";
const NAME = "apilot";

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

const platform = PLATFORM_MAP[process.platform];
const arch = ARCH_MAP[process.arch];

if (!platform || !arch) {
  console.error(`Unsupported platform: ${process.platform}-${process.arch}`);
  process.exit(1);
}

const isWindows = process.platform === "win32";
const ext = isWindows ? ".zip" : ".tar.gz";
const archiveName = `${NAME}-${VERSION}-${platform}-${arch}${ext}`;
const DOWNLOAD_URL = `https://github.com/${REPO}/releases/download/v${VERSION}/${archiveName}`;

const binDir = path.join(__dirname, "..", "bin");
const dest = path.join(binDir, NAME + (isWindows ? ".exe" : ""));

fs.mkdirSync(binDir, { recursive: true });

function download(url, destPath) {
  const sslFlag = isWindows ? "--ssl-revoke-best-effort " : "";
  execSync(
    `curl ${sslFlag}--fail --location --silent --show-error --connect-timeout 10 --max-time 120 --output "${destPath}" "${url}"`,
    { stdio: ["ignore", "ignore", "pipe"] }
  );
}

function install() {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "apilot-"));
  const archivePath = path.join(tmpDir, archiveName);

  try {
    console.log(`Downloading apilot v${VERSION} for ${platform}-${arch}...`);
    download(DOWNLOAD_URL, archivePath);

    if (isWindows) {
      execSync(
        `powershell -Command "Expand-Archive -Path '${archivePath}' -DestinationPath '${tmpDir}'"`,
        { stdio: "ignore" }
      );
    } else {
      execSync(`tar -xzf "${archivePath}" -C "${tmpDir}"`, { stdio: "ignore" });
    }

    const binaryName = NAME + (isWindows ? ".exe" : "");
    const extractedBinary = path.join(tmpDir, binaryName);

    fs.copyFileSync(extractedBinary, dest);
    fs.chmodSync(dest, 0o755);
    console.log(`apilot v${VERSION} installed successfully`);
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

try {
  install();
} catch (err) {
  console.error(`Failed to install apilot:`, err.message);
  console.error(
    `\nIf you are behind a firewall or in a restricted network, try setting a proxy:\n` +
    `  export https_proxy=http://your-proxy:port\n` +
    `  npm install -g @tangcent/apilot`
  );
  process.exit(1);
}
