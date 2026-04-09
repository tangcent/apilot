import * as vscode from 'vscode';

export type OutputDestination = 'channel' | 'file';
export type Formatter = 'markdown' | 'curl' | 'postman';

export interface Settings {
  formatter: Formatter;
  outputDestination: OutputDestination;
  outputFile: string;
  binaryPath: string;
}

export function getSettings(): Settings {
  const cfg = vscode.workspace.getConfiguration('apilot');
  return {
    formatter: cfg.get<Formatter>('formatter', 'markdown'),
    outputDestination: cfg.get<OutputDestination>('outputDestination', 'channel'),
    outputFile: cfg.get<string>('outputFile', ''),
    binaryPath: cfg.get<string>('binaryPath', ''),
  };
}
