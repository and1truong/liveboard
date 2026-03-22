package mcp

import (
	"net/http"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/workspace"
)

// MCPServer wraps an MCP server with LiveBoard tools.
type MCPServer struct {
	server *mcpsdk.Server
	ws     *workspace.Workspace
	eng    *board.Engine
}

// New creates an MCPServer with all LiveBoard tools registered.
func New(ws *workspace.Workspace, eng *board.Engine, version string) *MCPServer {
	s := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "liveboard",
		Version: version,
	}, nil)

	m := &MCPServer{server: s, ws: ws, eng: eng}
	m.registerBoardTools()
	m.registerCardTools()
	m.registerColumnTools()
	return m
}

// Server returns the underlying MCP server.
func (m *MCPServer) Server() *mcpsdk.Server { return m.server }

// StreamableHTTPHandler returns an http.Handler for the Streamable HTTP transport.
func (m *MCPServer) StreamableHTTPHandler() http.Handler {
	return mcpsdk.NewStreamableHTTPHandler(
		func(_ *http.Request) *mcpsdk.Server { return m.server },
		nil,
	)
}
