package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/takashabe/gco-o11y-mcp/internal/server"
)

func main() {
	var (
		transportType = flag.String("transport", "stdio", "Transport type: stdio or streamable-http")
		httpAddr      = flag.String("addr", ":8080", "HTTP address for streamable-http transport")
		serverName    = flag.String("name", "gcp-o11y-mcp", "Server name")
		serverVersion = flag.String("version", "1.0.0", "Server version")
	)
	flag.Parse()

	// サーバー設定
	config := server.Config{
		ServerName:    *serverName,
		ServerVersion: *serverVersion,
		TransportType: *transportType,
		HTTPAddr:      *httpAddr,
	}

	// サーバーを作成
	mcpServer, err := server.NewGCPObservabilityMCPServer(config)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// コンテキストとシグナルハンドリング
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// シグナルハンドリング
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	// サーバー開始
	log.Printf("Starting GCP Observability MCP Server...")
	log.Printf("Transport: %s", *transportType)
	if *transportType == "streamable-http" {
		log.Printf("HTTP Address: %s", *httpAddr)
	}

	if err := mcpServer.Start(ctx); err != nil {
		log.Printf("Server stopped: %v", err)
	}

	// クリーンアップ
	if err := mcpServer.Stop(); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	log.Println("Server shutdown complete")
}
