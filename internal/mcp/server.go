package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/takashabe/gco-o11y-mcp/pkg/types"
)

type Server struct {
	tools map[string]Tool
}

type Tool interface {
	Name() string
	Description() string
	Schema() types.Schema
	Execute(args map[string]interface{}) (*types.CallToolResult, error)
}

func NewServer() *Server {
	return &Server{
		tools: make(map[string]Tool),
	}
}

func (s *Server) RegisterTool(tool Tool) {
	s.tools[tool.Name()] = tool
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, req.ID, -32700, "Parse error", nil)
		return
	}

	switch req.Method {
	case "tools/list":
		s.handleListTools(w, req)
	case "tools/call":
		s.handleCallTool(w, req)
	default:
		s.writeError(w, req.ID, -32601, "Method not found", nil)
	}
}

func (s *Server) handleListTools(w http.ResponseWriter, req types.JSONRPCRequest) {
	var tools []types.Tool
	for _, tool := range s.tools {
		tools = append(tools, types.Tool{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.Schema(),
		})
	}

	result := types.ListToolsResult{
		Tools: tools,
	}

	s.writeResponse(w, req.ID, result)
}

func (s *Server) handleCallTool(w http.ResponseWriter, req types.JSONRPCRequest) {
	var params types.CallToolParams
	paramBytes, err := json.Marshal(req.Params)
	if err != nil {
		s.writeError(w, req.ID, -32602, "Invalid params", nil)
		return
	}

	if err := json.Unmarshal(paramBytes, &params); err != nil {
		s.writeError(w, req.ID, -32602, "Invalid params", nil)
		return
	}

	tool, ok := s.tools[params.Name]
	if !ok {
		s.writeError(w, req.ID, -32602, fmt.Sprintf("Tool not found: %s", params.Name), nil)
		return
	}

	args, ok := params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	result, err := tool.Execute(args)
	if err != nil {
		log.Printf("Tool execution error: %v", err)
		s.writeError(w, req.ID, -32603, "Internal error", err.Error())
		return
	}

	s.writeResponse(w, req.ID, result)
}

func (s *Server) writeResponse(w http.ResponseWriter, id interface{}, result interface{}) {
	resp := types.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) writeError(w http.ResponseWriter, id interface{}, code int, message string, data interface{}) {
	resp := types.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &types.RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(resp)
}