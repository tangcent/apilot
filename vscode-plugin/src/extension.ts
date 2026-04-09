import * as vscode from 'vscode';
import * as path from 'path';
import { getSettings } from './settings';
import { runExport } from './runner';

export function activate(context: vscode.ExtensionContext): void {
  const cmd = vscode.commands.registerCommand('apilot.export', async (uri?: vscode.Uri) => {
    // Resolve source directory from right-click target or active editor
    let sourceDir: string | undefined;
    if (uri) {
      sourceDir = uri.fsPath;
    } else if (vscode.window.activeTextEditor) {
      sourceDir = path.dirname(vscode.window.activeTextEditor.document.uri.fsPath);
    }

    if (!sourceDir) {
      vscode.window.showWarningMessage('APilot: no source file or folder selected.');
      return;
    }

    const settings = getSettings();
    await runExport(sourceDir, settings);
  });

  context.subscriptions.push(cmd);
}

export function deactivate(): void {}
