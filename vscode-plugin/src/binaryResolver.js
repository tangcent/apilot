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
exports.resolveBinary = resolveBinary;
const path = __importStar(require("path"));
const fs = __importStar(require("fs"));
/**
 * Resolves the path to the apilot-cli binary.
 * Priority: user-configured custom path → bundled platform binary.
 */
function resolveBinary(settings) {
    if (settings.binaryPath) {
        return settings.binaryPath;
    }
    const platform = process.platform; // "linux" | "darwin" | "win32"
    const arch = process.arch; // "x64" | "arm64"
    const suffix = platform === 'win32' ? '.exe' : '';
    const name = `apilot-${platform}-${arch}${suffix}`;
    const bundledPath = path.join(__dirname, '..', 'bin', name);
    if (!fs.existsSync(bundledPath)) {
        throw new Error(`Bundled binary not found: ${bundledPath}\n` +
            `Platform: ${platform}/${arch}\n` +
            `Set 'apilot.binaryPath' in settings to point to a valid apilot-cli binary.`);
    }
    return bundledPath;
}
//# sourceMappingURL=binaryResolver.js.map