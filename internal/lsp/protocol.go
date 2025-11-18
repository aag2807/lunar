package lsp

// LSP Protocol types based on the Language Server Protocol specification

// Message represents a JSON-RPC message
type Message struct {
	JSONRPC string `json:"jsonrpc"`
}

// RequestMessage represents a JSON-RPC request
type RequestMessage struct {
	Message
	ID     interface{} `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// ResponseMessage represents a JSON-RPC response
type ResponseMessage struct {
	Message
	ID     interface{}    `json:"id"`
	Result interface{}    `json:"result,omitempty"`
	Error  *ResponseError `json:"error,omitempty"`
}

// NotificationMessage represents a JSON-RPC notification
type NotificationMessage struct {
	Message
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// ResponseError represents a JSON-RPC error
type ResponseError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error codes
const (
	ParseError           = -32700
	InvalidRequest       = -32600
	MethodNotFound       = -32601
	InvalidParams        = -32602
	InternalError        = -32603
	ServerNotInitialized = -32002
	UnknownErrorCode     = -32001
	RequestCancelled     = -32800
)

// InitializeParams represents the initialize request parameters
type InitializeParams struct {
	ProcessID             *int               `json:"processId"`
	RootURI               string             `json:"rootUri"`
	Capabilities          ClientCapabilities `json:"capabilities"`
	InitializationOptions interface{}        `json:"initializationOptions,omitempty"`
}

// ClientCapabilities represents client capabilities
type ClientCapabilities struct {
	TextDocument TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Workspace    WorkspaceClientCapabilities    `json:"workspace,omitempty"`
}

// TextDocumentClientCapabilities represents text document client capabilities
type TextDocumentClientCapabilities struct {
	Synchronization    *TextDocumentSyncClientCapabilities `json:"synchronization,omitempty"`
	Completion         *CompletionClientCapabilities       `json:"completion,omitempty"`
	Hover              *HoverClientCapabilities            `json:"hover,omitempty"`
	Definition         *DefinitionClientCapabilities       `json:"definition,omitempty"`
	PublishDiagnostics *PublishDiagnosticsCapabilities     `json:"publishDiagnostics,omitempty"`
}

// TextDocumentSyncClientCapabilities represents sync capabilities
type TextDocumentSyncClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	WillSave            bool `json:"willSave,omitempty"`
	WillSaveWaitUntil   bool `json:"willSaveWaitUntil,omitempty"`
	DidSave             bool `json:"didSave,omitempty"`
}

// CompletionClientCapabilities represents completion capabilities
type CompletionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// HoverClientCapabilities represents hover capabilities
type HoverClientCapabilities struct {
	DynamicRegistration bool     `json:"dynamicRegistration,omitempty"`
	ContentFormat       []string `json:"contentFormat,omitempty"`
}

// DefinitionClientCapabilities represents definition capabilities
type DefinitionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// PublishDiagnosticsCapabilities represents diagnostics capabilities
type PublishDiagnosticsCapabilities struct {
	RelatedInformation bool `json:"relatedInformation,omitempty"`
}

// WorkspaceClientCapabilities represents workspace capabilities
type WorkspaceClientCapabilities struct {
	ApplyEdit bool `json:"applyEdit,omitempty"`
}

// InitializeResult represents the initialize response
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

// ServerCapabilities represents server capabilities
type ServerCapabilities struct {
	TextDocumentSync           *TextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
	HoverProvider              bool                     `json:"hoverProvider,omitempty"`
	CompletionProvider         *CompletionOptions       `json:"completionProvider,omitempty"`
	DefinitionProvider         bool                     `json:"definitionProvider,omitempty"`
	ReferencesProvider         bool                     `json:"referencesProvider,omitempty"`
	DocumentSymbolProvider     bool                     `json:"documentSymbolProvider,omitempty"`
	WorkspaceSymbolProvider    bool                     `json:"workspaceSymbolProvider,omitempty"`
	DocumentFormattingProvider bool                     `json:"documentFormattingProvider,omitempty"`
	RenameProvider             bool                     `json:"renameProvider,omitempty"`
}

// TextDocumentSyncOptions represents text document sync options
type TextDocumentSyncOptions struct {
	OpenClose bool                 `json:"openClose,omitempty"`
	Change    TextDocumentSyncKind `json:"change,omitempty"`
	Save      *SaveOptions         `json:"save,omitempty"`
}

// TextDocumentSyncKind represents text document sync kind
type TextDocumentSyncKind int

const (
	SyncNone        TextDocumentSyncKind = 0
	SyncFull        TextDocumentSyncKind = 1
	SyncIncremental TextDocumentSyncKind = 2
)

// SaveOptions represents save options
type SaveOptions struct {
	IncludeText bool `json:"includeText,omitempty"`
}

// CompletionOptions represents completion options
type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
	ResolveProvider   bool     `json:"resolveProvider,omitempty"`
}

// TextDocumentIdentifier identifies a text document
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// VersionedTextDocumentIdentifier identifies a versioned text document
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

// TextDocumentItem represents a text document item
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// DidOpenTextDocumentParams represents didOpen parameters
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams represents didChange parameters
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// TextDocumentContentChangeEvent represents a content change event
type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength int    `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

// DidCloseTextDocumentParams represents didClose parameters
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DidSaveTextDocumentParams represents didSave parameters
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         string                 `json:"text,omitempty"`
}

// Position represents a position in a text document
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a range in a text document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location in a resource
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Diagnostic represents a diagnostic
type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity,omitempty"`
	Code     interface{}        `json:"code,omitempty"`
	Source   string             `json:"source,omitempty"`
	Message  string             `json:"message"`
}

// DiagnosticSeverity represents diagnostic severity
type DiagnosticSeverity int

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

// PublishDiagnosticsParams represents publishDiagnostics parameters
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// TextDocumentPositionParams represents text document position parameters
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// Hover represents hover information
type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// MarkupContent represents markup content
type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Markup kinds
const (
	PlainText = "plaintext"
	Markdown  = "markdown"
)

// CompletionParams represents completion parameters
type CompletionParams struct {
	TextDocumentPositionParams
	Context *CompletionContext `json:"context,omitempty"`
}

// CompletionContext represents completion context
type CompletionContext struct {
	TriggerKind      CompletionTriggerKind `json:"triggerKind"`
	TriggerCharacter string                `json:"triggerCharacter,omitempty"`
}

// CompletionTriggerKind represents completion trigger kind
type CompletionTriggerKind int

const (
	Invoked                         CompletionTriggerKind = 1
	TriggerCharacter                CompletionTriggerKind = 2
	TriggerForIncompleteCompletions CompletionTriggerKind = 3
)

// CompletionItem represents a completion item
type CompletionItem struct {
	Label         string             `json:"label"`
	Kind          CompletionItemKind `json:"kind,omitempty"`
	Detail        string             `json:"detail,omitempty"`
	Documentation interface{}        `json:"documentation,omitempty"`
	InsertText    string             `json:"insertText,omitempty"`
}

// CompletionItemKind represents completion item kind
type CompletionItemKind int

const (
	TextCompletion     CompletionItemKind = 1
	MethodCompletion   CompletionItemKind = 2
	FunctionCompletion CompletionItemKind = 3
	FieldCompletion    CompletionItemKind = 4
	VariableCompletion CompletionItemKind = 6
	ClassCompletion    CompletionItemKind = 7
	InterfaceKind      CompletionItemKind = 8
	ModuleCompletion   CompletionItemKind = 9
	PropertyCompletion CompletionItemKind = 10
	KeywordCompletion  CompletionItemKind = 14
	SnippetCompletion  CompletionItemKind = 15
	EnumCompletion     CompletionItemKind = 13
	EnumMemberKind     CompletionItemKind = 20
)

// CompletionList represents a list of completion items
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}
