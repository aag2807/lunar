// Lunar for VS Code: starts `lunar lsp` and connects it to Lunar buffers.
// With the project in WSL, this runs in the WSL extension host
// ("extensionKind": ["workspace"]), so lunar.serverPath is the Linux binary.
const path = require('path');
const vscode = require('vscode');
const { LanguageClient, TransportKind } = require('vscode-languageclient/node');

let client;

function serverOptions() {
    const exe = vscode.workspace.getConfiguration('lunar').get('serverPath', 'lunar');
    // The old lunar-lsp binary is the server itself; lunar needs `lsp`.
    const args = path.basename(exe).startsWith('lunar-lsp') ? [] : ['lsp'];
    return { command: exe, args, transport: TransportKind.stdio };
}

async function start() {
    client = new LanguageClient('lunar', 'Lunar', serverOptions(), {
        documentSelector: [{ scheme: 'file', language: 'lunar' }],
        synchronize: {
            fileEvents: vscode.workspace.createFileSystemWatcher('**/{*.lunar,lunar.config.json}'),
        },
    });
    try {
        await client.start();
    } catch (err) {
        const exe = serverOptions().command;
        vscode.window.showErrorMessage(
            `Lunar: could not start '${exe} lsp' (${err.message}). Install lunar or set lunar.serverPath.`);
    }
}

async function activate(context) {
    context.subscriptions.push(
        vscode.commands.registerCommand('lunar.restartServer', async () => {
            if (client) {
                await client.stop();
            }
            await start();
            vscode.window.showInformationMessage('Lunar language server restarted');
        }),
        vscode.workspace.onDidChangeConfiguration(async (e) => {
            if (e.affectsConfiguration('lunar.serverPath')) {
                await vscode.commands.executeCommand('lunar.restartServer');
            }
        }),
    );
    await start();
}

function deactivate() {
    return client ? client.stop() : undefined;
}

module.exports = { activate, deactivate };
