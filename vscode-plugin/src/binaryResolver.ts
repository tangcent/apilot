import * as path from 'path';
import * as fs from 'fs';
import { Settings } from './settings';

/**
 * Resolves the path to the apilot-cli binary.
 * Priority: user-configured custom path → bundled platform binary.
 */
export function resolveBinary(settings: Settings): string {
  if (settings.binaryPath) {
    return settings.binaryPath;
  }

  const platform = process.platform;  // "linux" | "darwin" | "win32"
  const arch = process.arch;          // "x64" | "arm64"
  const suffix = platform === 'win32' ? '.exe' : '';
  const name = `apilot-cli-${platform}-${arch}${suffix}`;
  const bundledPath = path.join(__dirname, '..', 'bin', name);

  if (!fs.existsSync(bundledPath)) {
    throw new Error(
      `Bundled binary not found: ${bundledPath}\n` +
      `Platform: ${platform}/${arch}\n` +
      `Set 'apilot.binaryPath' in settings to point to a valid apilot-cli binary.`
    );
  }

  return bundledPath;
}
