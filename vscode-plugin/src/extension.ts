import * as vscode from 'vscode';
import { getSettings } from './settings';
import { runExport } from './runner';
import { showExportDialog } from './exportDialog';

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

    const exportOptions = await showExportDialog({
      formatter: settings.formatter,
      format: settings.format,
      outputDestination: settings.outputDestination,
      outputFile: settings.outputFile,
    });

    if (!exportOptions) {
      return;
    }

    await runExport(sourcePath, settings, exportOptions);
  });

  context.subscriptions.push(cmd);
}

export function deactivate(): void {}
