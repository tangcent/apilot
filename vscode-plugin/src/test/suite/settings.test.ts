import * as assert from 'assert';
import * as vscode from 'vscode';

suite('settings', () => {
  test('getSettings returns default values', () => {
    const cfg = vscode.workspace.getConfiguration('apilot');
    const formatter = cfg.get<string>('formatter', 'markdown');
    const format = cfg.get<string>('format', 'simple');
    const outputDestination = cfg.get<string>('outputDestination', 'channel');
    const outputFile = cfg.get<string>('outputFile', '');
    const binaryPath = cfg.get<string>('binaryPath', '');

    assert.strictEqual(formatter, 'markdown');
    assert.strictEqual(format, 'simple');
    assert.strictEqual(outputDestination, 'channel');
    assert.strictEqual(outputFile, '');
    assert.strictEqual(binaryPath, '');
  });
});
