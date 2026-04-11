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

function getPlatformInfo(platform, arch, version) {
  version = version || VERSION;
  const mappedPlatform = PLATFORM_MAP[platform];
  const mappedArch = ARCH_MAP[arch];

  if (!mappedPlatform || !mappedArch) {
    return null;
  }

  const isWindows = platform === "win32";
  const ext = isWindows ? ".zip" : ".tar.gz";
  const archiveName = `${NAME}-${version}-${mappedPlatform}-${mappedArch}${ext}`;
  const downloadUrl = `https://github.com/${REPO}/releases/download/v${version}/${archiveName}`;
  const binaryName = NAME + (isWindows ? ".exe" : "");

  return {
    platform: mappedPlatform,
    arch: mappedArch,
    isWindows,
    ext,
    archiveName,
    downloadUrl,
    binaryName,
  };
}

function download(url, destPath, isWindows) {
  const sslFlag = isWindows ? "--ssl-revoke-best-effort " : "";
  execSync(
    `curl ${sslFlag}--fail --location --silent --show-error --connect-timeout 10 --max-time 120 --output "${destPath}" "${url}"`,
    { stdio: ["ignore", "ignore", "pipe"] }
  );
}

function install(info, opts) {
  opts = opts || {};
  const _download = opts.download || download;
  const _mkdirSync = opts.mkdirSync || fs.mkdirSync;
  const _mkdtempSync = opts.mkdtempSync || fs.mkdtempSync;
  const _copyFileSync = opts.copyFileSync || fs.copyFileSync;
  const _chmodSync = opts.chmodSync || fs.chmodSync;
  const _rmSync = opts.rmSync || fs.rmSync;
  const _execSync = opts.execSync || execSync;
  const _console = opts.console || console;

  const binDir = path.join(__dirname, "..", "bin");
  const dest = path.join(binDir, info.binaryName);

  _mkdirSync(binDir, { recursive: true });

  const tmpDir = _mkdtempSync(path.join(os.tmpdir(), "apilot-"));
  const archivePath = path.join(tmpDir, info.archiveName);

  try {
    _console.log(`Downloading apilot v${VERSION} for ${info.platform}-${info.arch}...`);
    _download(info.downloadUrl, archivePath, info.isWindows);

    if (info.isWindows) {
      _execSync(
        `powershell -Command "Expand-Archive -Path '${archivePath}' -DestinationPath '${tmpDir}'"`,
        { stdio: "ignore" }
      );
    } else {
      _execSync(`tar -xzf "${archivePath}" -C "${tmpDir}"`, { stdio: "ignore" });
    }

    const extractedBinary = path.join(tmpDir, info.binaryName);

    _copyFileSync(extractedBinary, dest);
    _chmodSync(dest, 0o755);
    _console.log(`apilot v${VERSION} installed successfully`);
  } finally {
    _rmSync(tmpDir, { recursive: true, force: true });
  }
}

function runInstall() {
  const info = getPlatformInfo(process.platform, process.arch);

  if (!info) {
    console.error(`Unsupported platform: ${process.platform}-${process.arch}`);
    process.exit(1);
  }

  try {
    install(info);
  } catch (err) {
    console.error(`Failed to install apilot:`, err.message);
    console.error(
      `\nIf you are behind a firewall or in a restricted network, try setting a proxy:\n` +
      `  export https_proxy=http://your-proxy:port\n` +
      `  npm install -g @tangcent/apilot`
    );
    process.exit(1);
  }
}

module.exports = { getPlatformInfo, download, install, PLATFORM_MAP, ARCH_MAP, NAME, REPO };

if (require.main === module) {
  runInstall();
}
