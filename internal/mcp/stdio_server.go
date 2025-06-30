package mcp

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/takashabe/gco-o11y-mcp/pkg/types"
)

type StdioServer struct {
	tools map[string]Tool
}

func NewStdioServer() *StdioServer {
	return &StdioServer{
		tools: make(map[string]Tool),
	}
}

func (s *StdioServer) RegisterTool(tool Tool) {
	s.tools[tool.Name()] = tool
}

func (s *StdioServer) HandleRequest(req interface{}) interface{} {
	_, ok := req.(map[string]interface{})
	if !ok {
		return s.createError(nil, -32700, "Parse error", nil)
	}

	var request types.JSONRPCRequest
	reqBytes, _ := json.Marshal(req)
	if err := json.Unmarshal(reqBytes, &request); err != nil {
		return s.createError(nil, -32700, "Parse error", nil)
	}

	switch request.Method {
	case "initialize":
		return s.handleInitialize(request)
	case "tools/list":
		return s.handleListTools(request)
	case "tools/call":
		return s.handleCallTool(request)
	case "notifications/initialized":
		// Just acknowledge the notification
		return nil
	default:
		return s.createError(request.ID, -32601, "Method not found", nil)
	}
}

func (s *StdioServer) handleInitialize(req types.JSONRPCRequest) interface{} {
	return types.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "0.1.0",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "gcp-o11y-mcp",
				"version": "0.1.0",
			},
		},
	}
}

func (s *StdioServer) handleListTools(req types.JSONRPCRequest) interface{} {
	var tools []types.Tool
	for _, tool := range s.tools {
		tools = append(tools, types.Tool{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.Schema(),
		})
	}

	return types.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: types.ListToolsResult{
			Tools: tools,
		},
	}
}

func (s *StdioServer) handleCallTool(req types.JSONRPCRequest) interface{} {
	var params types.CallToolParams
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		return s.createError(req.ID, -32602, "Invalid params", nil)
	}

	if err := json.Unmarshal(paramBytes, &params); err != nil {
		return s.createError(req.ID, -32602, "Invalid params", nil)
	}

	tool, ok := s.tools[params.Name]
	if !ok {
		return s.createError(req.ID, -32602, fmt.Sprintf("Tool not found: %s", params.Name), nil)
	}

	args, ok := params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	result, err := tool.Execute(args)
	if err != nil {
		log.Printf("Tool execution error: %v", err)
		return s.createError(req.ID, -32603, "Internal error", err.Error())
	}

	return types.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *StdioServer) createError(id interface{}, code int, message string, data interface{}) interface{} {
	return types.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &types.RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}