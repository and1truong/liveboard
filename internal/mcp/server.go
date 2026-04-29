package mcp

import (
	"net/http"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/and1truong/liveboard/internal/board"
	"github.com/and1truong/liveboard/internal/workspace"
)

// Server wraps an MCP server with LiveBoard tools.
type Server struct {
	server *mcpsdk.Server
	ws     *workspace.Workspace
	eng    *board.Engine
}

// New creates a Server with all LiveBoard tools registered.
func New(ws *workspace.Workspace, eng *board.Engine, version string) *Server {
	s := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "liveboard",
		Version: version,
	}, nil)

	m := &Server{server: s, ws: ws, eng: eng}
	m.registerBoardTools()
	m.registerCardTools()
	m.registerColumnTools()
	m.registerAttachmentTools()
	return m
}

// MCP returns the underlying MCP server.
func (m *Server) MCP() *mcpsdk.Server { return m.server }

// StreamableHTTPHandler returns an http.Handler for the Streamable HTTP transport.
func (m *Server) StreamableHTTPHandler() http.Handler {
	return mcpsdk.NewStreamableHTTPHandler(
		func(_ *http.Request) *mcpsdk.Server { return m.server },
		nil,
	)
}
