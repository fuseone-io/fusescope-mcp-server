package docs

import (
	"strings"

	mcp "github.com/fuseone-io/fusescope-mcp-server/internal/mcpcontract"
	"github.com/fuseone-io/fusescope-mcp-server/pkg/toolerrors"
)

const (
	CodeOutOfScopeURL  = toolerrors.CodeOutOfScopeURL
	CodeDocNotFound    = toolerrors.CodeDocNotFound
	CodeHeadingMissing = toolerrors.CodeHeadingMissing
	CodeIndexNotReady  = toolerrors.CodeIndexNotReady
)

func ToolError(code, message string, extra map[string]any) *mcp.CallToolResult {
	structured := map[string]any{"code": code}
	for k, v := range extra {
		structured[k] = v
	}
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: message},
		},
		StructuredContent: structured,
	}
}

func OutOfScopeURLError(url string) *mcp.CallToolResult {
	message := "URL is outside " + DocsRootURL + " scope: " + url
	if strings.HasPrefix(url, "fusescope://") {
		message += ". fusescope:// URIs are MCP resources: read them with resources/read, not fusescope_fetch_doc. To find a docs page, call fusescope_search_docs and pass one of its URLs here."
	}
	return ToolError(CodeOutOfScopeURL, message, nil)
}

func DocNotFoundError(url string) *mcp.CallToolResult {
	return ToolError(CodeDocNotFound, "Document is valid but not present in the docs index: "+url, nil)
}

func HeadingNotFoundError(heading string, headings []Heading) *mcp.CallToolResult {
	return ToolError(CodeHeadingMissing, "Heading not found in document: "+heading, map[string]any{
		"available_headings": headings,
	})
}

func IndexNotReadyError() *mcp.CallToolResult {
	return ToolError(CodeIndexNotReady, "Docs index is not ready yet; retry shortly.", map[string]any{
		"retry_after_seconds": 5,
	})
}
