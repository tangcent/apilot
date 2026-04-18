"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.runExport = runExport;
const cp = __importStar(require("child_process"));
const vscode = __importStar(require("vscode"));
const binaryResolver_1 = require("./binaryResolver");
const outputChannel = vscode.window.createOutputChannel('APilot');
/**
 * Runs apilot-cli against the given source directory and writes output
 * to the configured destination (VSCode channel or file).
 */
async function runExport(sourceDir, settings) {
    let binary;
    try {
        binary = (0, binaryResolver_1.resolveBinary)(settings);
    }
    catch (err) {
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
        proc.stdout.on('data', (chunk) => { stdout += chunk.toString(); });
        proc.stderr.on('data', (chunk) => { stderr += chunk.toString(); });
        proc.on('close', (code) => {
            if (code !== 0) {
                vscode.window.showErrorMessage(`APilot failed:\n${stderr}`);
            }
            else if (settings.outputDestination === 'channel') {
                outputChannel.clear();
                outputChannel.append(stdout);
                outputChannel.show();
            }
            else if (settings.outputDestination === 'file' && settings.outputFile) {
                vscode.window.showInformationMessage(`APilot: output written to ${settings.outputFile}`);
            }
            resolve();
        });
    });
}
//# sourceMappingURL=runner.js.map