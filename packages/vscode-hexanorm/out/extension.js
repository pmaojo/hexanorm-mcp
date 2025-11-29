"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.activate = activate;
exports.deactivate = deactivate;
const vscode = require("vscode");
function activate(context) {
    console.log('Hexanorm extension is now active!');
    let disposable = vscode.commands.registerCommand('hexanorm.start', () => {
        vscode.window.showInformationMessage('Hexanorm Server Started (Placeholder)');
        // Logic to start the MCP server binary would go here
    });
    context.subscriptions.push(disposable);
}
function deactivate() { }
//# sourceMappingURL=extension.js.map