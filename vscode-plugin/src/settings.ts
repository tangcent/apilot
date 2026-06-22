import * as vscode from 'vscode';

export type Formatter = 'markdown' | 'curl' | 'postman';
export type FormatVariant = 'simple' | 'detailed';
export type OutputDestination = 'channel' | 'file';

export interface Settings {
  formatter: Formatter;
  format: FormatVariant;
  outputDestination: OutputDestination;
  outputFile: string;
  binaryPath: string;
  postmanApiKey: string;
}

export interface ExportOptions {
  formatter: Formatter;
  format: FormatVariant;
  outputDestination: OutputDestination;
  outputFile: string;
}

export function getSettings(): Settings {
  const cfg = vscode.workspace.getConfiguration('apilot');
  return {
    formatter: cfg.get<Formatter>('formatter', 'markdown'),
    format: cfg.get<FormatVariant>('format', 'simple'),
    outputDestination: cfg.get<OutputDestination>('outputDestination', 'channel'),
    outputFile: cfg.get<string>('outputFile', ''),
    binaryPath: cfg.get<string>('binaryPath', ''),
    postmanApiKey: cfg.get<string>('postmanApiKey', ''),
  };
}
