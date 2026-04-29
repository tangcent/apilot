import * as assert from 'assert';
import * as vscode from 'vscode';

suite('extension', () => {
  test('apilot.export command is registered', async () => {
    const ext = vscode.extensions.getExtension('tangcent.apilot');
    if (ext && !ext.isActive) {
      await ext.activate();
    }
    const commands = await vscode.commands.getCommands(true);
    assert.ok(commands.includes('apilot.export'), 'apilot.export command should be registered');
  });
});
