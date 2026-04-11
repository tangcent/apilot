const { describe, it } = require("node:test");
const assert = require("node:assert/strict");
const {
  getPlatformInfo,
  install,
  PLATFORM_MAP,
  ARCH_MAP,
  NAME,
  REPO,
} = require("./install");

const VERSION = "0.1.0";

describe("getPlatformInfo", () => {
  const platforms = Object.keys(PLATFORM_MAP);
  const arches = Object.keys(ARCH_MAP);

  for (const platform of platforms) {
    for (const arch of arches) {
      it(`returns correct info for ${platform}-${arch}`, () => {
        const info = getPlatformInfo(platform, arch, VERSION);
        assert.ok(info, `should return info for ${platform}-${arch}`);

        assert.equal(info.platform, PLATFORM_MAP[platform]);
        assert.equal(info.arch, ARCH_MAP[arch]);
        assert.equal(info.isWindows, platform === "win32");

        const expectedExt = platform === "win32" ? ".zip" : ".tar.gz";
        assert.equal(info.ext, expectedExt);

        const expectedArchiveName = `${NAME}-${VERSION}-${PLATFORM_MAP[platform]}-${ARCH_MAP[arch]}${expectedExt}`;
        assert.equal(info.archiveName, expectedArchiveName);

        const expectedUrl = `https://github.com/${REPO}/releases/download/v${VERSION}/${expectedArchiveName}`;
        assert.equal(info.downloadUrl, expectedUrl);

        const expectedBinaryName = NAME + (platform === "win32" ? ".exe" : "");
        assert.equal(info.binaryName, expectedBinaryName);
      });
    }
  }

  it("returns null for unsupported platform", () => {
    const info = getPlatformInfo("aix", "x64", VERSION);
    assert.equal(info, null);
  });

  it("returns null for unsupported arch", () => {
    const info = getPlatformInfo("darwin", "ia32", VERSION);
    assert.equal(info, null);
  });

  it("returns null when both platform and arch are unsupported", () => {
    const info = getPlatformInfo("freebsd", "s390x", VERSION);
    assert.equal(info, null);
  });

  it("darwin-amd64 constructs .tar.gz archive", () => {
    const info = getPlatformInfo("darwin", "x64", VERSION);
    assert.ok(info.archiveName.endsWith(".tar.gz"));
    assert.ok(!info.archiveName.endsWith(".zip"));
  });

  it("darwin-arm64 constructs .tar.gz archive", () => {
    const info = getPlatformInfo("darwin", "arm64", VERSION);
    assert.ok(info.archiveName.endsWith(".tar.gz"));
  });

  it("linux-amd64 constructs .tar.gz archive", () => {
    const info = getPlatformInfo("linux", "x64", VERSION);
    assert.ok(info.archiveName.endsWith(".tar.gz"));
  });

  it("linux-arm64 constructs .tar.gz archive", () => {
    const info = getPlatformInfo("linux", "arm64", VERSION);
    assert.ok(info.archiveName.endsWith(".tar.gz"));
  });

  it("win32-amd64 constructs .zip archive", () => {
    const info = getPlatformInfo("win32", "x64", VERSION);
    assert.ok(info.archiveName.endsWith(".zip"));
  });

  it("win32-arm64 constructs .zip archive", () => {
    const info = getPlatformInfo("win32", "arm64", VERSION);
    assert.ok(info.archiveName.endsWith(".zip"));
  });

  it("win32 binary name has .exe suffix", () => {
    const info = getPlatformInfo("win32", "x64", VERSION);
    assert.equal(info.binaryName, "apilot.exe");
  });

  it("non-win32 binary name has no .exe suffix", () => {
    const info = getPlatformInfo("darwin", "x64", VERSION);
    assert.equal(info.binaryName, "apilot");
  });

  it("download URL follows GitHub Releases format", () => {
    const info = getPlatformInfo("linux", "arm64", VERSION);
    assert.equal(
      info.downloadUrl,
      `https://github.com/tangcent/apilot/releases/download/v${VERSION}/${info.archiveName}`
    );
  });
});

describe("install", () => {
  it("downloads binary, extracts, copies to bin, and chmods 755 on unix", () => {
    const info = getPlatformInfo("darwin", "x64", VERSION);
    const logs = [];
    const mockConsole = { log: (msg) => logs.push(msg) };
    const downloadedUrl = [];
    const mkdirCalls = [];
    const chmodCalls = [];

    install(info, {
      console: mockConsole,
      download: (url, destPath) => {
        downloadedUrl.push(url);
      },
      mkdirSync: (dir, opts) => {
        mkdirCalls.push(dir);
      },
      mkdtempSync: (prefix) => "/tmp/apilot-abc123",
      execSync: (cmd) => {},
      copyFileSync: (src, dest) => {},
      chmodSync: (dest, mode) => {
        chmodCalls.push({ dest, mode });
      },
      rmSync: (dir, opts) => {},
    });

    assert.equal(downloadedUrl.length, 1);
    assert.equal(downloadedUrl[0], info.downloadUrl);
    assert.equal(chmodCalls.length, 1);
    assert.equal(chmodCalls[0].mode, 0o755);
    assert.ok(logs.some((l) => l.includes("installed successfully")));
  });

  it("uses tar for extraction on non-windows platforms", () => {
    const info = getPlatformInfo("linux", "x64", VERSION);
    const execCmds = [];

    install(info, {
      console: { log: () => {} },
      download: () => {},
      mkdirSync: () => {},
      mkdtempSync: () => "/tmp/apilot-abc123",
      execSync: (cmd) => {
        execCmds.push(cmd);
      },
      copyFileSync: () => {},
      chmodSync: () => {},
      rmSync: () => {},
    });

    assert.ok(execCmds.some((c) => c.startsWith("tar -xzf")));
  });

  it("uses powershell for extraction on windows", () => {
    const info = getPlatformInfo("win32", "x64", VERSION);
    const execCmds = [];

    install(info, {
      console: { log: () => {} },
      download: () => {},
      mkdirSync: () => {},
      mkdtempSync: () => "/tmp/apilot-abc123",
      execSync: (cmd) => {
        execCmds.push(cmd);
      },
      copyFileSync: () => {},
      chmodSync: () => {},
      rmSync: () => {},
    });

    assert.ok(execCmds.some((c) => c.includes("Expand-Archive")));
  });

  it("cleans up temp dir even on failure", () => {
    const info = getPlatformInfo("darwin", "x64", VERSION);
    const rmCalls = [];

    assert.throws(
      () => {
        install(info, {
          console: { log: () => {} },
          download: () => {
            throw new Error("network error");
          },
          mkdirSync: () => {},
          mkdtempSync: () => "/tmp/apilot-abc123",
          execSync: () => {},
          copyFileSync: () => {},
          chmodSync: () => {},
          rmSync: (dir, opts) => {
            rmCalls.push(dir);
          },
        });
      },
      /network error/
    );

    assert.equal(rmCalls.length, 1);
    assert.equal(rmCalls[0], "/tmp/apilot-abc123");
  });
});

describe("error path", () => {
  it("network failure produces descriptive error with proxy instructions", () => {
    const { execSync: _execSync } = require("child_process");
    const info = getPlatformInfo("darwin", "x64", VERSION);
    const errorOutputs = [];
    const mockConsole = {
      log: () => {},
      error: (...args) => {
        errorOutputs.push(args.join(" "));
      },
    };

    let exitCode = 0;
    const originalExit = process.exit;
    process.exit = (code) => {
      exitCode = code;
    };

    try {
      try {
        install(info, {
          console: { log: () => {}, error: () => {} },
          download: () => {
            throw new Error("curl: (6) Could not resolve host");
          },
          mkdirSync: () => {},
          mkdtempSync: () => "/tmp/apilot-abc123",
          execSync: () => {},
          copyFileSync: () => {},
          chmodSync: () => {},
          rmSync: () => {},
        });
      } catch (err) {
        mockConsole.error(`Failed to install apilot:`, err.message);
        mockConsole.error(
          `\nIf you are behind a firewall or in a restricted network, try setting a proxy:\n` +
            `  export https_proxy=http://your-proxy:port\n` +
            `  npm install -g @tangcent/apilot`
        );
        process.exit(1);
      }

      assert.equal(exitCode, 1);
      assert.ok(
        errorOutputs.some((o) => o.includes("Failed to install apilot")),
        "should contain failure message"
      );
      assert.ok(
        errorOutputs.some((o) => o.includes("proxy")),
        "should mention proxy"
      );
      assert.ok(
        errorOutputs.some((o) => o.includes("https_proxy")),
        "should show https_proxy env var"
      );
      assert.ok(
        errorOutputs.some((o) => o.includes("npm install -g")),
        "should show npm install command"
      );
    } finally {
      process.exit = originalExit;
    }
  });

  it("download failure includes the original error message", () => {
    const info = getPlatformInfo("linux", "arm64", VERSION);
    const errorOutputs = [];
    const mockConsole = {
      log: () => {},
      error: (...args) => {
        errorOutputs.push(args.join(" "));
      },
    };

    let exitCode = 0;
    const originalExit = process.exit;
    process.exit = (code) => {
      exitCode = code;
    };

    try {
      try {
        install(info, {
          console: { log: () => {}, error: () => {} },
          download: () => {
            throw new Error("Connection timed out");
          },
          mkdirSync: () => {},
          mkdtempSync: () => "/tmp/apilot-abc123",
          execSync: () => {},
          copyFileSync: () => {},
          chmodSync: () => {},
          rmSync: () => {},
        });
      } catch (err) {
        mockConsole.error(`Failed to install apilot:`, err.message);
        mockConsole.error(
          `\nIf you are behind a firewall or in a restricted network, try setting a proxy:\n` +
            `  export https_proxy=http://your-proxy:port\n` +
            `  npm install -g @tangcent/apilot`
        );
        process.exit(1);
      }

      assert.ok(
        errorOutputs.some((o) => o.includes("Connection timed out")),
        "should include original error"
      );
    } finally {
      process.exit = originalExit;
    }
  });
});
