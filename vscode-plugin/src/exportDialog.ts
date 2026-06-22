import * as vscode from 'vscode';
import { ExportOptions, Formatter, FormatVariant, OutputDestination } from './settings';

interface FormatterQuickPickItem extends vscode.QuickPickItem {
  formatter: Formatter;
}

const FORMATTER_ITEMS: FormatterQuickPickItem[] = [
  { label: 'Markdown', description: 'API documentation in Markdown', formatter: 'markdown' },
  { label: 'cURL', description: 'cURL command snippets', formatter: 'curl' },
  { label: 'Postman', description: 'Postman collection JSON', formatter: 'postman' },
];

interface FormatVariantQuickPickItem extends vscode.QuickPickItem {
  variant: FormatVariant;
}

const FORMAT_VARIANT_ITEMS: FormatVariantQuickPickItem[] = [
  { label: 'Simple', description: 'Concise output', variant: 'simple' },
  { label: 'Detailed', description: 'Verbose output with extra info', variant: 'detailed' },
];

interface DestinationQuickPickItem extends vscode.QuickPickItem {
  destination: OutputDestination;
}

const DESTINATION_ITEMS: DestinationQuickPickItem[] = [
  { label: 'Output Channel', description: 'Show in VS Code output panel', destination: 'channel' },
  { label: 'File', description: 'Save to a file', destination: 'file' },
];

export async function showExportDialog(defaults: ExportOptions): Promise<ExportOptions | undefined> {
  const formatter = await pickFormatter(defaults.formatter);
  if (!formatter) {
    return undefined;
  }

  const format = await pickFormatVariant(defaults.format);
  if (!format) {
    return undefined;
  }

  const outputDestination = await pickOutputDestination(defaults.outputDestination);
  if (!outputDestination) {
    return undefined;
  }

  let outputFile = defaults.outputFile;
  if (outputDestination === 'file') {
    const chosen = await pickOutputFile(defaults.outputFile, formatter);
    if (!chosen) {
      return undefined;
    }
    outputFile = chosen;
  }

  return { formatter, format, outputDestination, outputFile };
}

async function pickFormatter(defaultFormatter: Formatter): Promise<Formatter | undefined> {
  const items = FORMATTER_ITEMS.map(item => ({
    ...item,
    picked: item.formatter === defaultFormatter,
  }));

  const result = await vscode.window.showQuickPick(items, {
    placeHolder: 'Select output format',
    title: 'APilot Export — Formatter',
  });

  return result?.formatter;
}

async function pickFormatVariant(defaultVariant: FormatVariant): Promise<FormatVariant | undefined> {
  const items = FORMAT_VARIANT_ITEMS.map(item => ({
    ...item,
    picked: item.variant === defaultVariant,
  }));

  const result = await vscode.window.showQuickPick(items, {
    placeHolder: 'Select format variant',
    title: 'APilot Export — Format Variant',
  });

  return result?.variant;
}

async function pickOutputDestination(defaultDest: OutputDestination): Promise<OutputDestination | undefined> {
  const items = DESTINATION_ITEMS.map(item => ({
    ...item,
    picked: item.destination === defaultDest,
  }));

  const result = await vscode.window.showQuickPick(items, {
    placeHolder: 'Select output destination',
    title: 'APilot Export — Destination',
  });

  return result?.destination;
}

async function pickOutputFile(defaultPath: string, formatter: Formatter): Promise<string | undefined> {
  const filters: Record<string, string[]> = {};
  if (formatter === 'postman') {
    filters['Postman Collection'] = ['json'];
  } else if (formatter === 'markdown') {
    filters['Markdown'] = ['md'];
  } else {
    filters['All Files'] = ['*'];
  }
  filters['All Files'] = ['*'];

  const uri = await vscode.window.showSaveDialog({
    defaultUri: defaultPath ? vscode.Uri.file(defaultPath) : undefined,
    saveLabel: 'Export',
    title: 'APilot Export — Choose Output File',
    filters,
  });

  return uri?.fsPath;
}
