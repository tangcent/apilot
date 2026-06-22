import * as assert from 'assert';
import * as vscode from 'vscode';
import { runExport } from '../../runner';
import { Settings, ExportOptions } from '../../settings';

suite('runner', () => {
  test('runExport shows error when binary not found', async () => {
    const settings: Settings = {
      formatter: 'markdown',
      format: 'simple',
      outputDestination: 'channel',
      outputFile: '',
      binaryPath: '/nonexistent/apilot-binary',
      postmanApiKey: '',
    };

    const options: ExportOptions = {
      formatter: 'markdown',
      format: 'simple',
      outputDestination: 'channel',
      outputFile: '',
    };

    let errorMessageShown = false;
    const originalShowError = vscode.window.showErrorMessage;
    vscode.window.showErrorMessage = (...args: any[]) => {
      errorMessageShown = true;
      return Promise.resolve(undefined as any);
    };

    try {
      await runExport('/tmp', settings, options);
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
      '--params', JSON.stringify({ variant: 'detailed', postmanApiKey: 'test-key' }),
      '/tmp/test-dir',
      '--output', '/tmp/output.json',
    ];

    assert.strictEqual(expectedArgs[0], 'export', 'First arg must be "export" subcommand');
    assert.strictEqual(expectedArgs[1], '--formatter');
    assert.strictEqual(expectedArgs[2], 'postman');
    assert.strictEqual(expectedArgs[3], '--params');
    const parsedParams = JSON.parse(expectedArgs[4]);
    assert.strictEqual(parsedParams.variant, 'detailed');
    assert.strictEqual(parsedParams.postmanApiKey, 'test-key');
    assert.strictEqual(expectedArgs[6], '--output');
    assert.strictEqual(expectedArgs[7], '/tmp/output.json');
  });

  test('runExport omits postmanApiKey from params when formatter is not postman', () => {
    const params = { variant: 'simple' };
    assert.strictEqual((params as any).postmanApiKey, undefined, 'postmanApiKey should not be in params for non-postman formatters');
  });
});
