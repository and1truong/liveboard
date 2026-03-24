// Package mcp provides a Model Context Protocol server for LiveBoard.
package mcp

import (
	"encoding/json"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func errResult(err error) (*mcpsdk.CallToolResult, any, error) {
	r := &mcpsdk.CallToolResult{IsError: true}
	r.Content = []mcpsdk.Content{&mcpsdk.TextContent{Text: err.Error()}}
	return r, nil, nil
}

func jsonResult(v any) (*mcpsdk.CallToolResult, any, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errResult(err)
	}
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(data)}},
	}, nil, nil
}

func textResult(msg string) (*mcpsdk.CallToolResult, any, error) {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: msg}},
	}, nil, nil
}

func boolPtr(b bool) *bool { return &b }
