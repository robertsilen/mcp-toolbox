// Copyright 2025 Google LLC
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

package jsonrpc

import (
	"fmt"

	mcputil "github.com/googleapis/mcp-toolbox/internal/server/mcp/util"
)

// JSONRPC_VERSION is the version of JSON-RPC used by MCP.
const JSONRPC_VERSION = "2.0"

const (
	// Standard JSON-RPC error codes
	PARSE_ERROR                        = -32700
	INVALID_REQUEST                    = -32600
	METHOD_NOT_FOUND                   = -32601
	INVALID_PARAMS                     = -32602
	INTERNAL_ERROR                     = -32603
	HEADER_MISMATCH                    = -32001
	MISSING_REQUIRED_CLIENT_CAPABILITY = -32003

	// Custom auth error codes
	UNAUTHORIZED = -401
	FORBIDDEN    = -403
)

// ProgressToken is used to associate progress notifications with the original request.
type ProgressToken interface{}

// RequestId is a uniquely identifying ID for a request in JSON-RPC.
// It can be any JSON-serializable value, typically a number or string.
type RequestId interface{}

// Request represents a bidirectional message with method and parameters expecting a response.
type Request struct {
	Method string `json:"method"`
}

type RequestParams struct {
	Meta *RequestMetaObject `json:"_meta"`
}

type RequestMetaObject struct {
	// If specified, the caller is requesting out-of-band progress
	// notifications for this request (as represented by
	// notifications/progress). The value of this parameter is an
	// opaque token that will be attached to any subsequent
	// notifications. The receiver is not obligated to provide these
	// notifications.
	ProgressToken ProgressToken `json:"progressToken,omitempty"`
	/**
	 * The MCP Protocol Version being used for this request. Required.
	 *
	 * For the HTTP transport, this value MUST match the `MCP-Protocol-Version`
	 * header; otherwise the server MUST return a `400 Bad Request`. If the
	 * server does not support the requested version, it MUST return an
	 * UnsupportedProtocolVersionError.
	 */
	ProtocolVersion string `json:"io.modelcontextprotocol/protocolVersion"`
	/**
	 * Identifies the client software making the request. Required.
	 *
	 * The Implementation schema requires `name` and `version`; other
	 * fields are optional.
	 */
	ClientInfo Implementation `json:"io.modelcontextprotocol/clientInfo"`
	/**
	 * The client's capabilities for this specific request. Required.
	 *
	 * Capabilities are declared per-request rather than once at initialization;
	 * an empty object means the client supports no optional capabilities.
	 * Servers MUST NOT infer capabilities from prior requests.
	 */
	MetaClientCapabilities *ClientCapabilities `json:"io.modelcontextprotocol/clientCapabilities"`

	// W3C Trace Context fields for distributed tracing
	Traceparent string `json:"traceparent,omitempty"`
	Tracestate  string `json:"tracestate,omitempty"`
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

// ListChange represents whether the server supports notification for changes to the capabilities.
type ListChanged struct {
	ListChanged *bool `json:"listChanged,omitempty"`
}

// ClientCapabilities represents capabilities a client may support. Known
// capabilities are defined here, in this schema, but this is not a closed set: any
// client can define its own, additional capabilities.
type ClientCapabilities struct {
	// Experimental, non-standard capabilities that the client supports.
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	// Present if the client supports listing roots.
	Roots *ListChanged `json:"roots,omitempty"`
	// Present if the client supports sampling from an LLM.
	Sampling struct{} `json:"sampling,omitempty"`
}

// JSONRPCRequest represents a request that expects a response.
type JSONRPCRequest struct {
	Jsonrpc string    `json:"jsonrpc"`
	Id      RequestId `json:"id"`
	Request
	Params any `json:"params,omitempty"`
}

// Notification is a one-way message requiring no response.
type Notification struct {
	Method string `json:"method"`
	Params struct {
		Meta map[string]interface{} `json:"_meta,omitempty"`
	} `json:"params,omitempty"`
}

// JSONRPCNotification represents a notification which does not expect a response.
type JSONRPCNotification struct {
	Jsonrpc string `json:"jsonrpc"`
	Notification
}

// Result represents a response for the request query.
type Result struct {
	// This result property is reserved by the protocol to allow clients and
	// servers to attach additional metadata to their responses.
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

// JSONRPCResponse represents a successful (non-error) response to a request.
type JSONRPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Id      RequestId   `json:"id"`
	Result  interface{} `json:"result"`
}

// Error represents the error content.
type Error struct {
	// The error type that occurred.
	Code int `json:"code"`
	// A short description of the error. The message SHOULD be limited
	// to a concise single sentence.
	Message string `json:"message"`
	// Additional information about the error. The value of this member
	// is defined by the sender (e.g. detailed error information, nested errors etc.).
	Data interface{} `json:"data,omitempty"`
}

// String returns the error type as a string based on the error code.
func (e Error) String() string {
	switch e.Code {
	case METHOD_NOT_FOUND:
		return "method_not_found"
	case INVALID_PARAMS:
		return "invalid_params"
	case INTERNAL_ERROR:
		return "internal_error"
	case PARSE_ERROR:
		return "parse_error"
	case INVALID_REQUEST:
		return "invalid_request"
	case UNAUTHORIZED:
		return "unauthorized"
	case FORBIDDEN:
		return "forbidden"
	default:
		return "jsonrpc_error"
	}
}

// JSONRPCError represents a non-successful (error) response to a request.
type JSONRPCError struct {
	Jsonrpc string    `json:"jsonrpc"`
	Id      RequestId `json:"id"`
	Error   Error     `json:"error"`
}

// Generic baseMessage could either be a JSONRPCNotification or JSONRPCRequest
type BaseMessage struct {
	Jsonrpc string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Id      RequestId      `json:"id,omitempty"`
	Params  *RequestParams `json:"params,omitempty"`
}

// NewError is the standard JSONRPC response sent back when an error has been encountered.
func NewError(id RequestId, code int, message string, data any) JSONRPCError {
	return JSONRPCError{
		Jsonrpc: JSONRPC_VERSION,
		Id:      id,
		Error: Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func NewUnsupportedProtocolVersionError(id RequestId, v string) (JSONRPCError, error) {
	err := fmt.Errorf("unsupported protocol version")
	return JSONRPCError{
		Jsonrpc: JSONRPC_VERSION,
		Id:      id,
		Error: Error{
			Code:    INVALID_PARAMS,
			Message: err.Error(),
			Data: struct {
				Supported []string `json:"supported"`
				Requested string   `json:"requested"`
			}{
				Supported: mcputil.SUPPORTED_PROTOCOL_VERSIONS,
				Requested: v,
			},
		},
	}, err
}

func NewHeaderMismatchedError(id RequestId, err error) JSONRPCError {
	return JSONRPCError{
		Jsonrpc: JSONRPC_VERSION,
		Id:      id,
		Error: Error{
			Code:    HEADER_MISMATCH,
			Message: err.Error(),
		},
	}
}
