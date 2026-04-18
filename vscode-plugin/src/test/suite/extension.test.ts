import * as assert from 'assert';
import * as vscode from 'vscode';

suite('extension', () => {
  test('apilot.export command is registered', async () => {
    const commands = await vscode.commands.getCommands(true);
    assert.ok(commands.includes('apilot.export'), 'apilot.export command should be registered');
  });
});
