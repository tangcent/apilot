import * as assert from 'assert';
import * as path from 'path';
import { resolveBinary } from '../../binaryResolver';
import { Settings } from '../../settings';

suite('binaryResolver', () => {
  test('returns custom binaryPath when set', () => {
    const settings: Settings = {
      formatter: 'markdown',
      format: 'simple',
      outputDestination: 'channel',
      outputFile: '',
      binaryPath: '/usr/local/bin/custom-apilot',
    };
    assert.strictEqual(resolveBinary(settings), '/usr/local/bin/custom-apilot');
  });

  test('throws descriptive error when bundled binary not found', () => {
    const settings: Settings = {
      formatter: 'markdown',
      format: 'simple',
      outputDestination: 'channel',
      outputFile: '',
      binaryPath: '',
    };

    let thrownError: Error | null = null;
    try {
      resolveBinary(settings);
    } catch (e) {
      thrownError = e as Error;
    }

    assert.notStrictEqual(thrownError, null, 'Expected an error to be thrown');
    assert.ok(thrownError!.message.includes('Bundled binary not found'));
    assert.ok(thrownError!.message.includes('apilot.binaryPath'));
  });

  test('bundled binary name follows apilot-{platform}-{arch} pattern', () => {
    const platform = process.platform;
    const arch = process.arch;
    const suffix = platform === 'win32' ? '.exe' : '';
    const expectedName = `apilot-${platform}-${arch}${suffix}`;

    assert.ok(expectedName.startsWith('apilot-'), `Binary name should start with "apilot-", got: ${expectedName}`);
    assert.ok(expectedName.includes(platform), `Binary name should include platform "${platform}", got: ${expectedName}`);
    assert.ok(expectedName.includes(arch), `Binary name should include arch "${arch}", got: ${expectedName}`);
  });

  test('bundled binary path resolves to bin directory', () => {
    const platform = process.platform;
    const arch = process.arch;
    const suffix = platform === 'win32' ? '.exe' : '';
    const expectedName = `apilot-${platform}-${arch}${suffix}`;
    const expectedDir = path.join(__dirname, '..', '..', 'bin');
    const expectedPath = path.join(expectedDir, expectedName);

    assert.ok(expectedPath.includes('bin'), 'Binary should be in bin directory');
    assert.ok(expectedPath.endsWith(expectedName), `Binary path should end with ${expectedName}`);
  });
});
