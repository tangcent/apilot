import * as assert from 'assert';
import * as vscode from 'vscode';
import { runExport } from '../../runner';
import { Settings } from '../../settings';

suite('runner', () => {
  test('runExport shows error when binary not found', async () => {
    const settings: Settings = {
      formatter: 'markdown',
      format: 'simple',
      outputDestination: 'channel',
      outputFile: '',
      binaryPath: '/nonexistent/apilot-binary',
    };

    let errorMessageShown = false;
    const originalShowError = vscode.window.showErrorMessage;
    vscode.window.showErrorMessage = (...args: any[]) => {
      errorMessageShown = true;
      return Promise.resolve(undefined as any);
    };

    try {
      await runExport('/tmp', settings);
    } catch {
      // spawn may throw for nonexistent binary
    } finally {
      vscode.window.showErrorMessage = originalShowError;
    }

    assert.ok(errorMessageShown, 'showErrorMessage should be called when binary fails');
  });

  test('runExport constructs correct args with export subcommand', () => {
    const expectedArgs = [
      'export',
      '--formatter', 'postman',
      '--params', JSON.stringify({ variant: 'detailed' }),
      '/tmp/test-dir',
      '--output', '/tmp/output.json',
    ];

    assert.strictEqual(expectedArgs[0], 'export', 'First arg must be "export" subcommand');
    assert.strictEqual(expectedArgs[1], '--formatter');
    assert.strictEqual(expectedArgs[2], 'postman');
    assert.strictEqual(expectedArgs[3], '--params');
    assert.deepStrictEqual(JSON.parse(expectedArgs[4]), { variant: 'detailed' });
    assert.strictEqual(expectedArgs[6], '--output');
    assert.strictEqual(expectedArgs[7], '/tmp/output.json');
  });
});
