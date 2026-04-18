import * as vscode from 'vscode';

export type OutputDestination = 'channel' | 'file';
export type Formatter = 'markdown' | 'curl' | 'postman';
export type FormatVariant = 'simple' | 'detailed';

export interface Settings {
  formatter: Formatter;
  format: FormatVariant;
  outputDestination: OutputDestination;
  outputFile: string;
  binaryPath: string;
}

export function getSettings(): Settings {
  const cfg = vscode.workspace.getConfiguration('apilot');
  return {
    formatter: cfg.get<Formatter>('formatter', 'markdown'),
    format: cfg.get<FormatVariant>('format', 'simple'),
    outputDestination: cfg.get<OutputDestination>('outputDestination', 'channel'),
    outputFile: cfg.get<string>('outputFile', ''),
    binaryPath: cfg.get<string>('binaryPath', ''),
  };
}
