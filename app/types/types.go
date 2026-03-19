package types

import (
	"database/sql"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// Message represents a single chat message.
type Message struct {
	ID        int64  `json:"id"`
	ChatID    string `json:"chat_id"`
	Role      string `json:"role"` // "user" or "assistant"
	Content   string `json:"content"`
	SessionID int64  `json:"session_id"`
	CreatedAt int64  `json:"created_at"`
	WAMsgID   string `json:"wa_msg_id"`
	ReadAt    int64  `json:"read_at"`
}

// Summary represents an AI-generated session summary.
type Summary struct {
	ID        int64  `json:"id"`
	ChatID    string `json:"chat_id"`
	SessionID int64  `json:"session_id"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
}

// Contact represents a contact for the conversations list.
type Contact struct {
	ChatID      string `json:"chat_id"`
	LastMessage string `json:"last_message"`
	LastTime    int64  `json:"last_time"`
	SessionID   int64  `json:"session_id"`
}

// KnowledgeEntry represents a single knowledge base entry.
type KnowledgeEntry struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	Tags         string `json:"tags"`
	Compressed   string `json:"compressed"`
	Enabled      bool   `json:"enabled"`
	HasEmbedding bool   `json:"has_embedding"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

// LogEntry represents a single log record.
type LogEntry struct {
	ID        int64          `json:"id"`
	Event     string         `json:"event"`
	ChatID    string         `json:"chat_id,omitempty"`
	Data      map[string]any `json:"data"`
	CreatedAt int64          `json:"created_at"`
}

// LogFilter for querying logs from DB.
type LogFilter struct {
	Event  string
	ChatID string
	Since  int64
	Limit  int
	Offset int
}

// AuthUser represents a user in the users table.
type AuthUser struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuthSession holds session data for a logged-in user.
type AuthSession struct {
	UserID   int
	Username string
	Role     string
	Expiry   int64 // Unix timestamp
}

// GuardrailResult indicates whether a message is allowed or blocked.
type GuardrailResult struct {
	Allowed bool
	Reason  string // empty if allowed
	Reply   string // auto-reply if blocked (empty = silent drop); modified text if allowed but truncated
}

// PipelineResult holds the outcome of processing a message.
type PipelineResult struct {
	Response string
	Blocked  bool
	Reason   string
	Error    error
}

// GroupBasic holds minimal group info for the UI.
type GroupBasic struct {
	JID  string `json:"jid"`
	Name string `json:"name"`
}

// ToolHandler defines a single callable tool.
type ToolHandler struct {
	Definition openai.Tool
	Execute    func(chatID, args string) (string, error)
}

// ToolInfo is a simplified view of a tool for the web UI.
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CustomTool represents a user-defined API tool stored in SQLite.
type CustomTool struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Method       string `json:"method"`
	URLTemplate  string `json:"url_template"`
	Headers      string `json:"headers"`
	BodyTemplate string `json:"body_template"`
	Parameters   string `json:"parameters"`
	ResponsePath string `json:"response_path"`
	MaxBytes     int    `json:"max_bytes"`
	Enabled      int    `json:"enabled"`
	CreatedAt    int64  `json:"created_at"`
}

// CustomToolParam describes a parameter for a custom tool.
type CustomToolParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ExternalDatabase represents a user-configured external database stored in SQLite.
type ExternalDatabase struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Driver    string `json:"driver"` // "mysql" or "postgres"
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"`
	DBName    string `json:"dbname"`
	SSLMode   string `json:"ssl_mode"`
	MaxRows   int    `json:"max_rows"`
	Enabled   int    `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
}

// Agent represents a static AI agent with its own personality/config.
type Agent struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	SystemPrompt   string `json:"system_prompt"`
	UserPrompt     string `json:"user_prompt"`
	Model          string `json:"model"`
	MaxTokens      int    `json:"max_tokens"`
	BaseURL        string `json:"base_url"`
	APIKey         string `json:"api_key,omitempty"`
	Enabled        int    `json:"enabled"`
	IsDefault      int    `json:"is_default"`
	ChainTo        int64  `json:"chain_to"`
	ChainCondition string `json:"chain_condition"`
	RAGTags        string `json:"rag_tags"`
	// RAG
	RAGEnabled        bool   `json:"rag_enabled"`
	RAGMaxResults     int    `json:"rag_max_results"`
	RAGCompressed     bool   `json:"rag_compressed"`
	RAGMaxTokens      int    `json:"rag_max_tokens"`
	RAGEmbeddings     bool   `json:"rag_embeddings"`
	RAGEmbeddingModel string `json:"rag_embedding_model"`
	// Tools
	ToolsEnabled   bool `json:"tools_enabled"`
	ToolsMaxRounds int  `json:"tools_max_rounds"`
	ToolTimeoutSec int  `json:"tool_timeout_sec"`
	// Guardrails
	GuardMaxInput       int    `json:"guard_max_input"`
	GuardMaxOutput      int    `json:"guard_max_output"`
	GuardBlockedInput   string `json:"guard_blocked_input"`
	GuardBlockedOutput  string `json:"guard_blocked_output"`
	GuardPhoneList      string `json:"guard_phone_list"`
	GuardPhoneMode      string `json:"guard_phone_mode"`
	GuardBlockInjection bool   `json:"guard_block_injection"`
	GuardBlockPII       bool   `json:"guard_block_pii"`
	GuardBlockPIIPhone  bool   `json:"guard_block_pii_phone"`
	GuardBlockPIIEmail  bool   `json:"guard_block_pii_email"`
	GuardBlockPIICPF    bool   `json:"guard_block_pii_cpf"`
	KnowledgeExtract    bool   `json:"knowledge_extract"`
	CreatedAt           int64  `json:"created_at"`
}

// ExtDBConn holds an open connection to an external database.
type ExtDBConn struct {
	ID     int64
	Name   string
	Driver string
	Conn   *sql.DB
}

// MCPTool represents a discovered MCP tool mapped to OpenAI format.
type MCPTool struct {
	ServerName string // prefix for tool name
	Name       string // original MCP tool name
	FullName   string // "servername_toolname" registered in ToolRegistry
	Definition openai.Tool
}

// ReportResponse holds the response from a report query.
type ReportResponse struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
	Summary string     `json:"summary"`
}
