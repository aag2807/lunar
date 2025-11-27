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
	TextDocumentSync           *TextDocumentSyncOptions   `json:"textDocumentSync,omitempty"`
	HoverProvider              bool                       `json:"hoverProvider,omitempty"`
	CompletionProvider         *CompletionOptions         `json:"completionProvider,omitempty"`
	SignatureHelpProvider      *SignatureHelpOptions      `json:"signatureHelpProvider,omitempty"`
	DefinitionProvider         bool                       `json:"definitionProvider,omitempty"`
	ReferencesProvider         bool                       `json:"referencesProvider,omitempty"`
	DocumentSymbolProvider     bool                       `json:"documentSymbolProvider,omitempty"`
	WorkspaceSymbolProvider    bool                       `json:"workspaceSymbolProvider,omitempty"`
	DocumentFormattingProvider bool                       `json:"documentFormattingProvider,omitempty"`
	RenameProvider             bool                       `json:"renameProvider,omitempty"`
	CodeActionProvider         *CodeActionOptions         `json:"codeActionProvider,omitempty"`
	InlayHintProvider          *InlayHintOptions          `json:"inlayHintProvider,omitempty"`
	SemanticTokensProvider     *SemanticTokensOptions     `json:"semanticTokensProvider,omitempty"`
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

// SignatureHelpOptions represents signature help options
type SignatureHelpOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
	RetriggerCharacters []string `json:"retriggerCharacters,omitempty"`
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
	Label            string             `json:"label"`
	Kind             CompletionItemKind `json:"kind,omitempty"`
	Detail           string             `json:"detail,omitempty"`
	Documentation    interface{}        `json:"documentation,omitempty"`
	InsertText       string             `json:"insertText,omitempty"`
	InsertTextFormat InsertTextFormat   `json:"insertTextFormat,omitempty"`
	SortText         string             `json:"sortText,omitempty"`
	FilterText       string             `json:"filterText,omitempty"`
	Preselect        bool               `json:"preselect,omitempty"`
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

// InsertTextFormat represents the format of insert text
type InsertTextFormat int

const (
	PlainTextFormat InsertTextFormat = 1
	SnippetFormat   InsertTextFormat = 2
)

// SignatureHelpParams represents signature help parameters
type SignatureHelpParams struct {
	TextDocumentPositionParams
	Context *SignatureHelpContext `json:"context,omitempty"`
}

// SignatureHelpContext represents signature help context
type SignatureHelpContext struct {
	TriggerKind      SignatureHelpTriggerKind `json:"triggerKind"`
	TriggerCharacter string                   `json:"triggerCharacter,omitempty"`
	IsRetrigger      bool                     `json:"isRetrigger"`
	ActiveSignature  int                      `json:"activeSignatureHelp,omitempty"`
}

// SignatureHelpTriggerKind represents how signature help was triggered
type SignatureHelpTriggerKind int

const (
	SignatureHelpInvoked           SignatureHelpTriggerKind = 1
	SignatureHelpTriggerCharacter  SignatureHelpTriggerKind = 2
	SignatureHelpContentChange     SignatureHelpTriggerKind = 3
)

// SignatureHelp represents signature help
type SignatureHelp struct {
	Signatures      []SignatureInformation `json:"signatures"`
	ActiveSignature int                    `json:"activeSignature,omitempty"`
	ActiveParameter int                    `json:"activeParameter,omitempty"`
}

// SignatureInformation represents a function signature
type SignatureInformation struct {
	Label         string                 `json:"label"`
	Documentation interface{}            `json:"documentation,omitempty"`
	Parameters    []ParameterInformation `json:"parameters,omitempty"`
}

// ParameterInformation represents a function parameter
type ParameterInformation struct {
	Label         interface{} `json:"label"` // string or [int, int]
	Documentation interface{} `json:"documentation,omitempty"`
}

// DocumentSymbolParams represents document symbol parameters
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DocumentSymbol represents a symbol in a document
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// SymbolKind represents the kind of a symbol
type SymbolKind int

const (
	FileSymbol        SymbolKind = 1
	ModuleSymbol      SymbolKind = 2
	NamespaceSymbol   SymbolKind = 3
	PackageSymbol     SymbolKind = 4
	ClassSymbol       SymbolKind = 5
	MethodSymbol      SymbolKind = 6
	PropertySymbol    SymbolKind = 7
	FieldSymbol       SymbolKind = 8
	ConstructorSymbol SymbolKind = 9
	EnumSymbol        SymbolKind = 10
	InterfaceSymbol   SymbolKind = 11
	FunctionSymbol    SymbolKind = 12
	VariableSymbol    SymbolKind = 13
	ConstantSymbol    SymbolKind = 14
	StringSymbol      SymbolKind = 15
	NumberSymbol      SymbolKind = 16
	BooleanSymbol     SymbolKind = 17
	ArraySymbol       SymbolKind = 18
)

// WorkspaceSymbolParams represents workspace symbol parameters
type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}

// SymbolInformation represents a workspace symbol
type SymbolInformation struct {
	Name          string   `json:"name"`
	Kind          SymbolKind `json:"kind"`
	Location      Location `json:"location"`
	ContainerName string   `json:"containerName,omitempty"`
}

// ReferenceParams represents references parameters
type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

// ReferenceContext represents reference context
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// RenameParams represents rename parameters
type RenameParams struct {
	TextDocumentPositionParams
	NewName string `json:"newName"`
}

// WorkspaceEdit represents changes to many resources
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes,omitempty"`
}

// TextEdit represents a change to a text document
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// CodeActionOptions represents code action options
type CodeActionOptions struct {
	CodeActionKinds []string `json:"codeActionKinds,omitempty"`
}

// CodeActionParams represents code action parameters
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// CodeActionContext represents code action context
type CodeActionContext struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// CodeAction represents a code action
type CodeAction struct {
	Title       string         `json:"title"`
	Kind        string         `json:"kind,omitempty"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
	Edit        *WorkspaceEdit `json:"edit,omitempty"`
	Command     *Command       `json:"command,omitempty"`
}

// Command represents a command
type Command struct {
	Title     string        `json:"title"`
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// InlayHintOptions represents inlay hint options
type InlayHintOptions struct {
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// InlayHintParams represents inlay hint parameters
type InlayHintParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
}

// InlayHint represents an inlay hint
type InlayHint struct {
	Position     Position               `json:"position"`
	Label        string                 `json:"label"`
	Kind         int                    `json:"kind,omitempty"`
	PaddingLeft  bool                   `json:"paddingLeft,omitempty"`
	PaddingRight bool                   `json:"paddingRight,omitempty"`
	Tooltip      *MarkupContent         `json:"tooltip,omitempty"`
}

// InlayHint kinds
const (
	InlayHintKindType      = 1
	InlayHintKindParameter = 2
)

// SemanticTokensOptions represents semantic tokens options
type SemanticTokensOptions struct {
	Legend SemanticTokensLegend `json:"legend"`
	Range  bool                 `json:"range,omitempty"`
	Full   bool                 `json:"full,omitempty"`
}

// SemanticTokensLegend represents semantic tokens legend
type SemanticTokensLegend struct {
	TokenTypes     []string `json:"tokenTypes"`
	TokenModifiers []string `json:"tokenModifiers"`
}

// SemanticTokensParams represents semantic tokens parameters
type SemanticTokensParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// SemanticTokens represents semantic tokens
type SemanticTokens struct {
	Data []uint32 `json:"data"`
}

// Semantic token types
const (
	SemanticTokenTypeNamespace        = "namespace"
	SemanticTokenTypeType             = "type"
	SemanticTokenTypeClass            = "class"
	SemanticTokenTypeEnum             = "enum"
	SemanticTokenTypeInterface        = "interface"
	SemanticTokenTypeStruct           = "struct"
	SemanticTokenTypeTypeParameter    = "typeParameter"
	SemanticTokenTypeParameter        = "parameter"
	SemanticTokenTypeVariable         = "variable"
	SemanticTokenTypeProperty         = "property"
	SemanticTokenTypeEnumMember       = "enumMember"
	SemanticTokenTypeFunction         = "function"
	SemanticTokenTypeMethod           = "method"
	SemanticTokenTypeMacro            = "macro"
	SemanticTokenTypeKeyword          = "keyword"
	SemanticTokenTypeModifier         = "modifier"
	SemanticTokenTypeComment          = "comment"
	SemanticTokenTypeString           = "string"
	SemanticTokenTypeNumber           = "number"
	SemanticTokenTypeRegexp           = "regexp"
	SemanticTokenTypeOperator         = "operator"
)

// Semantic token modifiers
const (
	SemanticTokenModifierDeclaration    = "declaration"
	SemanticTokenModifierDefinition     = "definition"
	SemanticTokenModifierReadonly       = "readonly"
	SemanticTokenModifierStatic         = "static"
	SemanticTokenModifierDeprecated     = "deprecated"
	SemanticTokenModifierAbstract       = "abstract"
	SemanticTokenModifierAsync          = "async"
	SemanticTokenModifierModification   = "modification"
	SemanticTokenModifierDocumentation  = "documentation"
	SemanticTokenModifierDefaultLibrary = "defaultLibrary"
)

// Code action kinds
const (
	CodeActionKindQuickFix      = "quickfix"
	CodeActionKindRefactor      = "refactor"
	CodeActionKindRefactorExtract = "refactor.extract"
	CodeActionKindRefactorInline  = "refactor.inline"
	CodeActionKindRefactorRewrite = "refactor.rewrite"
	CodeActionKindSource          = "source"
	CodeActionKindSourceOrganizeImports = "source.organizeImports"
)
