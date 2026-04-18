import * as cp from 'child_process';
import * as vscode from 'vscode';
import { Settings } from './settings';
import { resolveBinary } from './binaryResolver';

const outputChannel = vscode.window.createOutputChannel('APilot');

/**
 * Runs apilot-cli against the given source directory and writes output
 * to the configured destination (VSCode channel or file).
 */
export async function runExport(sourceDir: string, settings: Settings): Promise<void> {
  let binary: string;
  try {
    binary = resolveBinary(settings);
  } catch (err: any) {
    vscode.window.showErrorMessage(err.message);
    return;
  }

  const args = [
    'export',
    '--formatter', settings.formatter,
    '--params', JSON.stringify({ variant: settings.format }),
    sourceDir,
  ];

  if (settings.outputDestination === 'file' && settings.outputFile) {
    args.push('--output', settings.outputFile);
  }

  return new Promise((resolve) => {
    const proc = cp.spawn(binary, args);
    let stdout = '';
    let stderr = '';

    proc.stdout.on('data', (chunk: Buffer) => { stdout += chunk.toString(); });
    proc.stderr.on('data', (chunk: Buffer) => { stderr += chunk.toString(); });

    proc.on('close', (code) => {
      if (code !== 0) {
        vscode.window.showErrorMessage(`APilot failed:\n${stderr}`);
      } else if (settings.outputDestination === 'channel') {
        outputChannel.clear();
        outputChannel.append(stdout);
        outputChannel.show();
      } else if (settings.outputDestination === 'file' && settings.outputFile) {
        vscode.window.showInformationMessage(`APilot: output written to ${settings.outputFile}`);
      }
      resolve();
    });
  });
}
