/**
 * Example VSCode Extension for Lunar Language Server
 *
 * This is a minimal example of how to create a VSCode extension
 * that integrates the Lunar LSP.
 *
 * To use:
 * 1. Create a new VSCode extension project
 * 2. Replace extension.ts with this file
 * 3. Install dependencies: npm install vscode-languageclient
 * 4. Build and run
 */

import * as path from 'path';
import { workspace, ExtensionContext } from 'vscode';

import {
    LanguageClient,
    LanguageClientOptions,
    ServerOptions,
    TransportKind
} from 'vscode-languageclient/node';

let client: LanguageClient;

export function activate(context: ExtensionContext) {
    // Path to the language server binary
    const serverPath = workspace.getConfiguration('lunar').get<string>('languageServer.path')
        || 'lunar-lsp';

    // Server options
    const serverOptions: ServerOptions = {
        command: serverPath,
        args: [],
        transport: TransportKind.stdio
    };

    // Client options
    const clientOptions: LanguageClientOptions = {
        // Register the server for lunar files
        documentSelector: [{ scheme: 'file', language: 'lunar' }],

        synchronize: {
            // Notify the server about file changes to '.lunar files contained in the workspace
            fileEvents: workspace.createFileSystemWatcher('**/*.lunar')
        },

        // Initialization options
        initializationOptions: {},
    };

    // Create the language client
    client = new LanguageClient(
        'lunarLanguageServer',
        'Lunar Language Server',
        serverOptions,
        clientOptions
    );

    // Start the client (this will also launch the server)
    client.start();

    console.log('Lunar Language Server started');
}

export function deactivate(): Thenable<void> | undefined {
    if (!client) {
        return undefined;
    }
    return client.stop();
}
