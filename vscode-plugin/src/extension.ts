import * as vscode from 'vscode';
import { getSettings } from './settings';
import { runExport } from './runner';

export function activate(context: vscode.ExtensionContext): void {
  const cmd = vscode.commands.registerCommand('apilot.export', async (uri?: vscode.Uri) => {
    let sourcePath: string | undefined;
    if (uri) {
      sourcePath = uri.fsPath;
    } else if (vscode.window.activeTextEditor) {
      sourcePath = vscode.window.activeTextEditor.document.uri.fsPath;
    }

    if (!sourcePath) {
      vscode.window.showWarningMessage('APilot: no source file or folder selected.');
      return;
    }

    const settings = getSettings();
    await runExport(sourcePath, settings);
  });

  context.subscriptions.push(cmd);
}

export function deactivate(): void {}
