package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/logging/logadmin"
	"google.golang.org/api/iterator"

	"github.com/takashabe/gco-o11y-mcp/pkg/types"
)

type ListLogEntriesTools struct {
	client      *Client
	cache       *LogCache
	rateLimiter *RateLimiter
}

type ListLogEntriesArgs struct {
	Filter   string `json:"filter,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
	OrderBy  string `json:"orderBy,omitempty"`
}

type LogEntry struct {
	Timestamp   string                 `json:"timestamp"`
	Severity    string                 `json:"severity"`
	LogName     string                 `json:"logName"`
	Resource    map[string]interface{} `json:"resource"`
	Labels      map[string]string      `json:"labels,omitempty"`
	TextPayload string                 `json:"textPayload,omitempty"`
	JSONPayload map[string]interface{} `json:"jsonPayload,omitempty"`
	InsertID    string                 `json:"insertId,omitempty"`
	TraceID     string                 `json:"traceId,omitempty"`
}

func NewListLogEntriesTools(client *Client) *ListLogEntriesTools {
	return &ListLogEntriesTools{
		client:      client,
		cache:       NewLogCache(),
		rateLimiter: NewRateLimiter(),
	}
}

func (t *ListLogEntriesTools) Name() string {
	return "list_log_entries"
}

func (t *ListLogEntriesTools) Description() string {
	return "List log entries from Google Cloud Logging. Supports filtering by timestamp, severity, resource, and custom filters."
}

func (t *ListLogEntriesTools) Schema() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"filter": {
				Type: "string",
			},
			"pageSize": {
				Type: "integer",
			},
			"orderBy": {
				Type: "string",
			},
		},
		AdditionalProperties: false,
	}
}

func (t *ListLogEntriesTools) Execute(args map[string]interface{}) (*types.CallToolResult, error) {
	var params ListLogEntriesArgs
	if argsBytes, err := json.Marshal(args); err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	} else if err := json.Unmarshal(argsBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	if params.PageSize == 0 {
		params.PageSize = 10
	}
	// Quota optimization: limit maximum page size
	if params.PageSize > 20 {
		params.PageSize = 20
	}

	if params.OrderBy == "" {
		params.OrderBy = "timestamp desc"
	}

	// Check cache first
	cacheKey := t.cache.GenerateKey(params)
	if cachedEntries, found := t.cache.Get(cacheKey); found {
		log.Printf("Cache hit for filter: %s", params.Filter)
		entriesJSON, err := json.MarshalIndent(cachedEntries, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cached entries: %w", err)
		}
		return &types.CallToolResult{
			Content: []types.Content{{
				Type: "text",
				Text: string(entriesJSON),
			}},
		}, nil
	}

	ctx := context.Background()
	var entries []LogEntry
	var err error
	
	// Execute with rate limiting and backoff
	err = t.rateLimiter.ExecuteWithBackoff(ctx, func() error {
		entries, err = t.listLogEntries(ctx, params)
		return err
	})
	if err != nil {
		log.Printf("Failed to list log entries: %v", err)
		return &types.CallToolResult{
			Content: []types.Content{{
				Type: "text",
				Text: fmt.Sprintf("Error listing log entries: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Cache the results (TTL: 2 minutes)
	t.cache.Set(cacheKey, entries, 2*time.Minute)

	entriesJSON, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal log entries: %w", err)
	}

	return &types.CallToolResult{
		Content: []types.Content{{
			Type: "text",
			Text: string(entriesJSON),
		}},
	}, nil
}

func (t *ListLogEntriesTools) listLogEntries(ctx context.Context, params ListLogEntriesArgs) ([]LogEntry, error) {
	client := t.client.LogAdminClient()
	
	// Add timeout to prevent long-running queries
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	iter := client.Entries(ctxWithTimeout,
		logadmin.Filter(params.Filter),
		logadmin.NewestFirst(),
	)

	var entries []LogEntry
	count := 0

	for {
		if count >= params.PageSize {
			break
		}

		entry, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate log entries: %w", err)
		}

		logEntry := LogEntry{
			Timestamp: entry.Timestamp.Format(time.RFC3339),
			Severity:  entry.Severity.String(),
			LogName:   entry.LogName,
			InsertID:  entry.InsertID,
			TraceID:   entry.Trace,
		}

		if entry.Resource != nil {
			logEntry.Resource = map[string]interface{}{
				"type":   entry.Resource.Type,
				"labels": entry.Resource.Labels,
			}
		}

		if entry.Labels != nil {
			logEntry.Labels = entry.Labels
		}

		switch payload := entry.Payload.(type) {
		case string:
			logEntry.TextPayload = payload
		case map[string]interface{}:
			logEntry.JSONPayload = payload
		}

		entries = append(entries, logEntry)
		count++
	}

	return entries, nil
}