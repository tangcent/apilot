import * as cp from 'child_process';
import * as vscode from 'vscode';
import { Settings, ExportOptions } from './settings';
import { resolveBinary } from './binaryResolver';

const outputChannel = vscode.window.createOutputChannel('APilot');

export async function runExport(sourcePath: string, settings: Settings, options: ExportOptions): Promise<void> {
  let binary: string;
  try {
    binary = resolveBinary(settings);
  } catch (err: any) {
    vscode.window.showErrorMessage(err.message);
    return;
  }

  const params: Record<string, any> = { variant: options.format };
  if (options.formatter === 'postman' && settings.postmanApiKey) {
    params.postmanApiKey = settings.postmanApiKey;
  }

  const args = [
    'export',
    '--formatter', options.formatter,
    '--params', JSON.stringify(params),
    sourcePath,
  ];

  if (options.outputDestination === 'file' && options.outputFile) {
    args.push('--output', options.outputFile);
  }

  return new Promise((resolve) => {
    const proc = cp.spawn(binary, args);
    let stdout = '';
    let stderr = '';

    proc.stdout.on('data', (chunk: Buffer) => { stdout += chunk.toString(); });
    proc.stderr.on('data', (chunk: Buffer) => { stderr += chunk.toString(); });

    proc.on('error', (err: Error) => {
      vscode.window.showErrorMessage(`APilot failed: ${err.message}`);
      resolve();
    });

    proc.on('close', (code) => {
      if (code !== 0) {
        vscode.window.showErrorMessage(`APilot failed:\n${stderr}`);
      } else if (options.outputDestination === 'channel') {
        outputChannel.clear();
        outputChannel.append(stdout);
        outputChannel.show();
      } else if (options.outputDestination === 'file' && options.outputFile) {
        vscode.window.showInformationMessage(`APilot: output written to ${options.outputFile}`);
      }
      resolve();
    });
  });
}
