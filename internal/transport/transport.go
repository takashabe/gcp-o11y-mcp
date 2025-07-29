package transport

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Transport はMCPサーバーの通信方式を抽象化するインターフェース
// stdio, Streamable HTTP などの異なる通信方式に対応可能
type Transport interface {
	// Connect はクライアントとの接続を確立し、セッションを開始する
	Connect(ctx context.Context, server *mcp.Server) error
	// Close は接続を閉じる
	Close() error
	// Type は通信方式の種類を返す
	Type() string
}

// StdioTransport はstdin/stdoutを使用したMCP通信を実装
type StdioTransport struct {
	server *mcp.Server
}

// NewStdioTransport は新しいStdioTransportを作成
func NewStdioTransport() *StdioTransport {
	return &StdioTransport{}
}

// Connect はstdin/stdoutを使用して接続を確立
func (t *StdioTransport) Connect(ctx context.Context, server *mcp.Server) error {
	t.server = server
	transport := mcp.NewStdioTransport()
	return server.Run(ctx, transport)
}

// Close は接続を閉じる（stdioの場合は特に処理なし）
func (t *StdioTransport) Close() error {
	return nil
}

// Type は通信方式の種類を返す
func (t *StdioTransport) Type() string {
	return "stdio"
}

// StreamableHTTPTransport は将来のStreamable HTTP対応用の構造体
// 現在は実装されていないが、インターフェースは統一
type StreamableHTTPTransport struct {
	server *mcp.Server
	addr   string
}

// NewStreamableHTTPTransport は新しいStreamableHTTPTransportを作成
func NewStreamableHTTPTransport(addr string) *StreamableHTTPTransport {
	return &StreamableHTTPTransport{
		addr: addr,
	}
}

// Connect はHTTPサーバーを起動して接続を確立
func (t *StreamableHTTPTransport) Connect(ctx context.Context, server *mcp.Server) error {
	// TODO: Streamable HTTPの実装
	// 現在はサポートされていないため、エラーを返す
	return ErrStreamableHTTPNotImplemented
}

// Close はHTTPサーバーを停止
func (t *StreamableHTTPTransport) Close() error {
	// TODO: Streamable HTTPの実装
	return nil
}

// Type は通信方式の種類を返す
func (t *StreamableHTTPTransport) Type() string {
	return "streamable-http"
}
