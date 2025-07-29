package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/takashabe/gco-o11y-mcp/pkg/types"
)

type PresetQueryTool struct {
	client      *Client
	cache       *LogCache
	rateLimiter *RateLimiter
}

type PresetQueryArgs struct {
	QueryName  string   `json:"queryName"`
	Parameters []string `json:"parameters,omitempty"`
}

func NewPresetQueryTool(client *Client) *PresetQueryTool {
	return &PresetQueryTool{
		client:      client,
		cache:       NewLogCache(),
		rateLimiter: NewRateLimiter(),
	}
}

func (t *PresetQueryTool) Name() string {
	return "preset_query"
}

func (t *PresetQueryTool) Description() string {
	return "Execute predefined optimized queries for common use cases like Cloud Run errors, recent logs, etc."
}

func (t *PresetQueryTool) Schema() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"queryName": {
				Type: "string",
			},
			"parameters": {
				Type: "array",
				Items: &types.Schema{
					Type: "string",
				},
			},
		},
		Required:             []string{"queryName"},
		AdditionalProperties: false,
	}
}

func (t *PresetQueryTool) Execute(args map[string]interface{}) (*types.CallToolResult, error) {
	var params PresetQueryArgs
	if argsBytes, err := json.Marshal(args); err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	} else if err := json.Unmarshal(argsBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	if params.QueryName == "" {
		return &types.CallToolResult{
			Content: []types.Content{{
				Type: "text",
				Text: "Error: queryName parameter is required",
			}},
			IsError: true,
		}, nil
	}

	// Check cache first
	cacheKey := t.cache.GenerateKey(params)
	if cachedEntries, found := t.cache.Get(cacheKey); found {
		log.Printf("Cache hit for preset query: %s", params.QueryName)
		result := map[string]interface{}{
			"queryName": params.QueryName,
			"count":     len(cachedEntries),
			"entries":   cachedEntries,
			"cached":    true,
		}
		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cached results: %w", err)
		}
		return &types.CallToolResult{
			Content: []types.Content{{
				Type: "text",
				Text: string(resultJSON),
			}},
		}, nil
	}

	// Get preset query
	filter, pageSize, err := GetPresetQuery(params.QueryName, params.Parameters...)
	if err != nil {
		return &types.CallToolResult{
			Content: []types.Content{{
				Type: "text",
				Text: fmt.Sprintf("Error: %v", err),
			}},
			IsError: true,
		}, nil
	}

	ctx := context.Background()
	var entries []LogEntry

	// Execute with rate limiting and backoff
	err = t.rateLimiter.ExecuteWithBackoff(ctx, func() error {
		entries, err = t.executePresetQuery(ctx, filter, pageSize)
		return err
	})

	if err != nil {
		log.Printf("Failed to execute preset query: %v", err)
		return &types.CallToolResult{
			Content: []types.Content{{
				Type: "text",
				Text: fmt.Sprintf("Error executing preset query: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Cache the results
	t.cache.Set(cacheKey, entries, 2*60*1000000000) // 2 minutes

	result := map[string]interface{}{
		"queryName": params.QueryName,
		"filter":    filter,
		"count":     len(entries),
		"entries":   entries,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal preset query results: %w", err)
	}

	return &types.CallToolResult{
		Content: []types.Content{{
			Type: "text",
			Text: string(resultJSON),
		}},
	}, nil
}

func (t *PresetQueryTool) executePresetQuery(ctx context.Context, filter string, pageSize int) ([]LogEntry, error) {
	// Use the same logic as list_log_entries but with preset parameters
	listTool := NewListLogEntriesTools(t.client)
	return listTool.listLogEntries(ctx, ListLogEntriesArgs{
		Filter:   filter,
		PageSize: pageSize,
		OrderBy:  "timestamp desc",
	})
}
