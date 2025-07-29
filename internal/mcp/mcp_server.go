package mcp

import (
	"fmt"
	"log"

	"github.com/takashabe/gco-o11y-mcp/pkg/types"
)

type Tool interface {
	Name() string
	Description() string
	Schema() types.Schema
	Execute(args map[string]interface{}) (*types.CallToolResult, error)
}

type MCPServer struct {
	tools map[string]Tool
}

func NewMCPServer() *MCPServer {
	return &MCPServer{
		tools: make(map[string]Tool),
	}
}

func (s *MCPServer) RegisterTool(tool Tool) {
	s.tools[tool.Name()] = tool
}

func (s *MCPServer) HandleRequest(request map[string]interface{}) interface{} {
	method, ok := request["method"].(string)
	if !ok {
		return s.createErrorResponse(nil, -32600, "Invalid Request", "Missing method")
	}

	id := request["id"]

	switch method {
	case "initialize":
		return s.handleInitialize(id, request)
	case "tools/list":
		return s.handleListTools(id)
	case "tools/call":
		return s.handleCallTool(id, request)
	case "notifications/initialized":
		// Notification - no response needed
		return nil
	default:
		return s.createErrorResponse(id, -32601, "Method not found", fmt.Sprintf("Unknown method: %s", method))
	}
}

func (s *MCPServer) handleInitialize(id interface{}, request map[string]interface{}) interface{} {
	log.Printf("Handling initialize request")

	params, _ := request["params"].(map[string]interface{})
	var clientInfo map[string]interface{}
	if params != nil {
		clientInfo, _ = params["clientInfo"].(map[string]interface{})
	}

	log.Printf("Client info: %v", clientInfo)

	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "gcp-o11y-mcp",
				"version": "1.0.0",
			},
		},
	}
}

func (s *MCPServer) handleListTools(id interface{}) interface{} {
	log.Printf("Handling tools/list request")

	var tools []map[string]interface{}
	for _, tool := range s.tools {
		toolDef := map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"inputSchema": s.convertSchemaToMap(tool.Schema()),
		}
		tools = append(tools, toolDef)
	}

	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]interface{}{
			"tools": tools,
		},
	}
}

func (s *MCPServer) handleCallTool(id interface{}, request map[string]interface{}) interface{} {
	params, ok := request["params"].(map[string]interface{})
	if !ok {
		return s.createErrorResponse(id, -32602, "Invalid params", "Missing params")
	}

	name, ok := params["name"].(string)
	if !ok {
		return s.createErrorResponse(id, -32602, "Invalid params", "Missing tool name")
	}

	log.Printf("Calling tool: %s", name)

	tool, exists := s.tools[name]
	if !exists {
		return s.createErrorResponse(id, -32602, "Tool not found", fmt.Sprintf("Tool '%s' not found", name))
	}

	arguments, _ := params["arguments"].(map[string]interface{})
	if arguments == nil {
		arguments = make(map[string]interface{})
	}

	log.Printf("Tool arguments: %v", arguments)

	result, err := tool.Execute(arguments)
	if err != nil {
		log.Printf("Tool execution error: %v", err)
		return s.createErrorResponse(id, -32603, "Internal error", err.Error())
	}

	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  s.convertCallToolResultToMap(result),
	}
}

func (s *MCPServer) convertSchemaToMap(schema types.Schema) map[string]interface{} {
	result := map[string]interface{}{
		"type": schema.Type,
	}

	if schema.Properties != nil {
		props := make(map[string]interface{})
		for name, prop := range schema.Properties {
			props[name] = s.convertSchemaToMap(prop)
		}
		result["properties"] = props
	}

	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}

	if schema.Items != nil {
		result["items"] = s.convertSchemaToMap(*schema.Items)
	}

	if schema.AdditionalProperties != nil {
		result["additionalProperties"] = schema.AdditionalProperties
	}

	return result
}

func (s *MCPServer) convertCallToolResultToMap(result *types.CallToolResult) map[string]interface{} {
	content := make([]map[string]interface{}, len(result.Content))
	for i, c := range result.Content {
		content[i] = map[string]interface{}{
			"type": c.Type,
			"text": c.Text,
		}
	}

	response := map[string]interface{}{
		"content": content,
	}

	if result.IsError {
		response["isError"] = true
	}

	return response
}

func (s *MCPServer) createErrorResponse(id interface{}, code int, message string, data interface{}) interface{} {
	errorObj := map[string]interface{}{
		"code":    code,
		"message": message,
	}

	if data != nil {
		errorObj["data"] = data
	}

	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   errorObj,
	}
}
