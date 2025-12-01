const vscode = require('vscode');
const { LanguageClient, TransportKind } = require('vscode-languageclient/node');

let client;

function activate(context) {
    const config = vscode.workspace.getConfiguration('lunar');
    const serverPath = config.get('serverPath', 'lunar-lsp');

    const serverOptions = {
        command: serverPath,
        args: [],
        transport: TransportKind.stdio
    };

    const clientOptions = {
        documentSelector: [
            { scheme: 'file', language: 'lunar' }
        ],
        synchronize: {
            fileEvents: [
                vscode.workspace.createFileSystemWatcher('**/*.lunar'),
                vscode.workspace.createFileSystemWatcher('**/*.d.lunar')
            ]
        }
    };

    client = new LanguageClient(
        'lunar',
        'Lunar Language Server',
        serverOptions,
        clientOptions
    );

    client.start();

    context.subscriptions.push(
        vscode.commands.registerCommand('lunar.restartServer', async () => {
            await client.stop();
            client.start();
            vscode.window.showInformationMessage('Lunar language server restarted');
        })
    );
}

function deactivate() {
    if (client) {
        return client.stop();
    }
}

module.exports = { activate, deactivate };
