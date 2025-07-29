package server

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/takashabe/gco-o11y-mcp/internal/logging"
	"github.com/takashabe/gco-o11y-mcp/internal/transport"
)

// GCPObservabilityMCPServer はGCP観測性データ用のMCPサーバー
type GCPObservabilityMCPServer struct {
	server        *mcp.Server
	transport     transport.Transport
	loggingClient *logging.Client
}

// Config はサーバーの設定
type Config struct {
	ServerName    string
	ServerVersion string
	TransportType string
	HTTPAddr      string // Streamable HTTPで使用
}

// NewGCPObservabilityMCPServer は新しいサーバーインスタンスを作成
func NewGCPObservabilityMCPServer(config Config) (*GCPObservabilityMCPServer, error) {
	// MCPサーバーを作成
	impl := &mcp.Implementation{
		Name:    config.ServerName,
		Version: config.ServerVersion,
	}
	server := mcp.NewServer(impl, nil)

	// Cloud Loggingクライアントを初期化
	ctx := context.Background()
	loggingClient, err := logging.NewClient(ctx, "")
	if err != nil {
		return nil, err
	}

	// 適切なトランスポートを選択
	var tp transport.Transport
	switch config.TransportType {
	case "stdio":
		tp = transport.NewStdioTransport()
	case "streamable-http":
		tp = transport.NewStreamableHTTPTransport(config.HTTPAddr)
	default:
		tp = transport.NewStdioTransport() // デフォルトはstdio
	}

	s := &GCPObservabilityMCPServer{
		server:        server,
		transport:     tp,
		loggingClient: loggingClient,
	}

	// ツールを登録
	s.registerTools()

	return s, nil
}

// registerTools は利用可能なツールを登録
func (s *GCPObservabilityMCPServer) registerTools() {
	// Preset Query Tool
	presetTool := logging.NewPresetQueryTool(s.loggingClient)
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        presetTool.Name(),
		Description: presetTool.Description(),
	}, s.createPresetQueryHandler(presetTool))

	// List Log Entries Tool
	listTool := logging.NewListLogEntriesTools(s.loggingClient)
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        listTool.Name(),
		Description: listTool.Description(),
	}, s.createListLogEntriesHandler(listTool))

	// Search Logs Tool
	searchTool := logging.NewSearchLogsTool(s.loggingClient)
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        searchTool.Name(),
		Description: searchTool.Description(),
	}, s.createSearchLogsHandler(searchTool))
}

// Start はサーバーを開始
func (s *GCPObservabilityMCPServer) Start(ctx context.Context) error {
	log.Printf("Starting GCP Observability MCP Server with transport: %s", s.transport.Type())
	return s.transport.Connect(ctx, s.server)
}

// Stop はサーバーを停止
func (s *GCPObservabilityMCPServer) Stop() error {
	return s.transport.Close()
}

// createPresetQueryHandler はPreset Query Tool用のハンドラーを作成
func (s *GCPObservabilityMCPServer) createPresetQueryHandler(tool *logging.PresetQueryTool) mcp.ToolHandlerFor[logging.PresetQueryArgs, any] {
	return func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[logging.PresetQueryArgs]) (*mcp.CallToolResultFor[any], error) {
		// 既存のツールのExecuteメソッドを呼び出し
		args := map[string]interface{}{
			"queryName":  params.Arguments.QueryName,
			"parameters": params.Arguments.Parameters,
		}

		result, err := tool.Execute(args)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil
		}

		// types.CallToolResultからmcp.CallToolResultForに変換
		var content []mcp.Content
		for _, c := range result.Content {
			content = append(content, &mcp.TextContent{Text: c.Text})
		}

		return &mcp.CallToolResultFor[any]{
			Content: content,
			IsError: result.IsError,
		}, nil
	}
}

// createListLogEntriesHandler はList Log Entries Tool用のハンドラーを作成
func (s *GCPObservabilityMCPServer) createListLogEntriesHandler(tool *logging.ListLogEntriesTools) mcp.ToolHandlerFor[logging.ListLogEntriesArgs, any] {
	return func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[logging.ListLogEntriesArgs]) (*mcp.CallToolResultFor[any], error) {
		// 既存のツールのExecuteメソッドを呼び出し
		args := map[string]interface{}{
			"filter":   params.Arguments.Filter,
			"pageSize": params.Arguments.PageSize,
			"orderBy":  params.Arguments.OrderBy,
		}

		result, err := tool.Execute(args)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil
		}

		// types.CallToolResultからmcp.CallToolResultForに変換
		var content []mcp.Content
		for _, c := range result.Content {
			content = append(content, &mcp.TextContent{Text: c.Text})
		}

		return &mcp.CallToolResultFor[any]{
			Content: content,
			IsError: result.IsError,
		}, nil
	}
}

// createSearchLogsHandler はSearch Logs Tool用のハンドラーを作成
func (s *GCPObservabilityMCPServer) createSearchLogsHandler(tool *logging.SearchLogsTool) mcp.ToolHandlerFor[logging.SearchLogsArgs, any] {
	return func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[logging.SearchLogsArgs]) (*mcp.CallToolResultFor[any], error) {
		// 既存のツールのExecuteメソッドを呼び出し
		args := map[string]interface{}{
			"query":    params.Arguments.Query,
			"severity": params.Arguments.Severity,
			"pageSize": params.Arguments.PageSize,
		}

		result, err := tool.Execute(args)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil
		}

		// types.CallToolResultからmcp.CallToolResultForに変換
		var content []mcp.Content
		for _, c := range result.Content {
			content = append(content, &mcp.TextContent{Text: c.Text})
		}

		return &mcp.CallToolResultFor[any]{
			Content: content,
			IsError: result.IsError,
		}, nil
	}
}
