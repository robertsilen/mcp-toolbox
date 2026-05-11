// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vdraft

import (
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/server/mcp/jsonrpc"
	mcputil "github.com/googleapis/mcp-toolbox/internal/server/mcp/util"
	"github.com/googleapis/mcp-toolbox/internal/tools"
)

// SERVER_NAME is the server name used in Implementation.
const SERVER_NAME = "Toolbox"

// PROTOCOL_VERSION is the version of the MCP protocol in this package.
const PROTOCOL_VERSION = mcputil.VERSION_DRAFT

// methods that are supported.
const (
	SERVER_DISCOVER = "server/discover"
	TOOLS_LIST      = "tools/list"
	TOOLS_CALL      = "tools/call"
	PROMPTS_LIST    = "prompts/list"
	PROMPTS_GET     = "prompts/get"
)

/* Discovery */

/**
 * A request from the client asking the server to advertise its supported
 * protocol versions, capabilities, and other metadata. Servers **MUST**
 * implement `server/discover`. Clients **MAY** call it but are not required
 * to — version negotiation can also happen inline via per-request `_meta`.
 */
type DiscoverRequest struct {
	jsonrpc.Request
	Params jsonrpc.RequestParams `json:"params,omitempty"`
}

// The result returned by the server for a {@link DiscoverRequest | server/discover} request.
type DiscoverResult struct {
	jsonrpc.Result
	/**
	 * MCP Protocol Versions this server supports. The client should choose a
	 * version from this list for use in subsequent requests.
	 */
	SupportedVersions []string `json:"supportedVersions"`
	/**
	 * The capabilities of the server.
	 */
	Capabilities ServerCapabilities `json:"capabilities"`
	/**
	 * Information about the server software implementation.
	 */
	ServerInfo Implementation `json:"serverInfo"`
	/**
	 * Natural-language guidance describing the server and its features.
	 *
	 * This can be used by clients to improve an LLM's understanding of
	 * available tools (e.g., by including it in a system prompt). It should
	 * focus on information that helps the model use the server effectively
	 * and should not duplicate information already in tool descriptions.
	 */
	Instructions string `json:"instructions,omitempty"`
}

// Base interface for metadata with name (identifier) and title (display name) properties.
type BaseMetadata struct {
	// Intended for programmatic or logical use, but used as a display name in past specs
	// or fallback (if title isn't present).
	Name string `json:"name"`
	// Intended for UI and end-user contexts — optimized to be human-readable and easily understood,
	//even by those unfamiliar with domain-specific terminology.
	//
	// If not provided, the name should be used for display (except for Tool,
	// where `annotations.title` should be given precedence over using `name`,
	// if present).
	Title string `json:"title,omitempty"`
}

// Implementation describes the name and version of an MCP implementation.
type Implementation struct {
	BaseMetadata
	Version string `json:"version"`
}

// ServerCapabilities represents capabilities that a server may support. Known
// capabilities are defined here, in this schema, but this is not a closed set: any
// server can define its own, additional capabilities.
type ServerCapabilities struct {
	Tools   *ListChanged `json:"tools,omitempty"`
	Prompts *ListChanged `json:"prompts,omitempty"`
}

// ListChange represents whether the server supports notification for changes to the capabilities.
type ListChanged struct {
	ListChanged *bool `json:"listChanged,omitempty"`
}

/* Empty result */

// EmptyResult represents a response that indicates success but carries no data.
type EmptyResult jsonrpc.Result

/* Pagination */

// Cursor is an opaque token used to represent a cursor for pagination.
type Cursor string

// Common params for paginated requests.
type PaginatedRequest struct {
	jsonrpc.Request
	Params PaginatedRequestParams `json:"params,omitempty"`
}

type PaginatedRequestParams struct {
	jsonrpc.RequestParams
	// An opaque token representing the current pagination position.
	// If provided, the server should return results starting after this cursor.
	Cursor Cursor `json:"cursor,omitempty"`
}

type PaginatedResult struct {
	jsonrpc.Result
	// An opaque token representing the pagination position after the last returned result.
	// If present, there may be more results available.
	NextCursor Cursor `json:"nextCursor,omitempty"`
}

/* Tools */

// Sent from the client to request a list of tools the server has.
type ListToolsRequest struct {
	PaginatedRequest
}

// The server's response to a tools/list request from the client.
type ListToolsResult struct {
	PaginatedResult
	Tools []tools.McpManifest `json:"tools"`
}

// Used by the client to invoke a tool provided by the server.
type CallToolRequest struct {
	jsonrpc.Request
	Params CallToolRequestParams `json:"params,omitempty"`
}

// Parameters for a `tools/call` request.
type CallToolRequestParams struct {
	jsonrpc.RequestParams
	/**
	 * The name of the tool.
	 */
	Name string `json:"name"`
	/**
	 * Arguments to use for the tool call.
	 */
	Arguments map[string]any `json:"arguments,omitempty"`
}

// The sender or recipient of messages and data in a conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Base for objects that include optional annotations for the client.
// The client can use annotations to inform how objects are used or displayed
type Annotated struct {
	Annotations *struct {
		// Describes who the intended customer of this object or data is.
		// It can include multiple entries to indicate content useful for multiple
		// audiences (e.g., `["user", "assistant"]`).
		Audience []Role `json:"audience,omitempty"`
		// Describes how important this data is for operating the server.
		//
		// A value of 1 means "most important," and indicates that the data is
		// effectively required, while 0 means "least important," and indicates that
		// the data is entirely optional.
		//
		// @TJS-type number
		// @minimum 0
		// @maximum 1
		Priority float64 `json:"priority,omitempty"`
	} `json:"annotations,omitempty"`
}

// TextContent represents text provided to or from an LLM.
type TextContent struct {
	Annotated
	Type string `json:"type"`
	// The text content of the message.
	Text string `json:"text"`
}

// The server's response to a tool call.
//
// Any errors that originate from the tool SHOULD be reported inside the result
// object, with `isError` set to true, _not_ as an MCP protocol-level error
// response. Otherwise, the LLM would not be able to see that an error occurred
// and self-correct.
//
// However, any errors in _finding_ the tool, an error indicating that the
// server does not support tool calls, or any other exceptional conditions,
// should be reported as an MCP error response.
type CallToolResult struct {
	jsonrpc.Result
	// Could be either a TextContent, ImageContent, or EmbeddedResources
	// For Toolbox, we will only be sending TextContent
	Content []TextContent `json:"content"`
	// Whether the tool call ended in an error.
	// If not set, this is assumed to be false (the call was successful).
	//
	// Any errors that originate from the tool SHOULD be reported inside the result
	// object, with `isError` set to true, _not_ as an MCP protocol-level error
	// response. Otherwise, the LLM would not be able to see that an error occurred
	// and self-correct.
	//
	// However, any errors in _finding_ the tool, an error indicating that the
	// server does not support tool calls, or any other exceptional conditions,
	// should be reported as an MCP error response.
	IsError bool `json:"isError,omitempty"`
	// An optional JSON object that represents the structured result of the tool call.
	StructuredContent map[string]any `json:"structuredContent,omitempty"`
}

// Additional properties describing a Tool to clients.
//
// NOTE: all properties in ToolAnnotations are **hints**.
// They are not guaranteed to provide a faithful description of
// tool behavior (including descriptive properties like `title`).
//
// Clients should never make tool use decisions based on ToolAnnotations
// received from untrusted servers.
type ToolAnnotations struct {
	// A human-readable title for the tool.
	Title string `json:"title,omitempty"`
	// If true, the tool does not modify its environment.
	// Default: false
	ReadOnlyHint bool `json:"readOnlyHint,omitempty"`
	// If true, the tool may perform destructive updates to its environment.
	// If false, the tool performs only additive updates.
	// (This property is meaningful only when `readOnlyHint == false`)
	// Default: true
	DestructiveHint bool `json:"destructiveHint,omitempty"`
	// If true, calling the tool repeatedly with the same arguments
	// will have no additional effect on the its environment.
	// (This property is meaningful only when `readOnlyHint == false`)
	// Default: false
	IdempotentHint bool `json:"idempotentHint,omitempty"`
	// If true, this tool may interact with an "open world" of external
	// entities. If false, the tool's domain of interaction is closed.
	// For example, the world of a web search tool is open, whereas that
	// of a memory tool is not.
	// Default: true
	OpenWorldHint bool `json:"openWorldHint,omitempty"`
}

/* Prompts */

// Sent from the client to request a list of prompts the server has.
type ListPromptsRequest struct {
	PaginatedRequest
}

// The server's response to a prompts/list request from the client.
type ListPromptsResult struct {
	PaginatedResult
	Prompts []prompts.McpManifest `json:"prompts"`
}

// Used by the client to get a prompt provided by the server.
type GetPromptRequest struct {
	jsonrpc.Request
	Params GetPromptRequestParams `json:"params"`
}

// Parameters for a `prompts/get` request.
type GetPromptRequestParams struct {
	jsonrpc.RequestParams
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// The server's response to a prompts/get request from the client.
type GetPromptResult struct {
	jsonrpc.Result
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// Describes a message returned as part of a prompt.
type PromptMessage struct {
	Role    string      `json:"role"`
	Content TextContent `json:"content"`
}
