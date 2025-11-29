import * as vscode from 'vscode';

export function activate(context: vscode.ExtensionContext) {
	console.log('Hexanorm extension is now active!');

	let disposable = vscode.commands.registerCommand('hexanorm.start', () => {
		vscode.window.showInformationMessage('Hexanorm Server Started (Placeholder)');
        // Logic to start the MCP server binary would go here
	});

	context.subscriptions.push(disposable);
}

export function deactivate() {}
