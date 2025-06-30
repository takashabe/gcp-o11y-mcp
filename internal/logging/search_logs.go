package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"google.golang.org/api/iterator"

	"github.com/takashabe/gco-o11y-mcp/pkg/types"
)

type SearchLogsTool struct {
	client      *Client
	cache       *LogCache
	rateLimiter *RateLimiter
}

type SearchLogsArgs struct {
	Query      string `json:"query"`
	StartTime  string `json:"startTime,omitempty"`
	EndTime    string `json:"endTime,omitempty"`
	Severity   string `json:"severity,omitempty"`
	Resource   string `json:"resource,omitempty"`
	LogName    string `json:"logName,omitempty"`
	PageSize   int    `json:"pageSize,omitempty"`
	OrderBy    string `json:"orderBy,omitempty"`
}

func NewSearchLogsTool(client *Client) *SearchLogsTool {
	return &SearchLogsTool{
		client:      client,
		cache:       NewLogCache(),
		rateLimiter: NewRateLimiter(),
	}
}

func (t *SearchLogsTool) Name() string {
	return "search_logs"
}

func (t *SearchLogsTool) Description() string {
	return "Search log entries with advanced filtering options including text query, time range, severity level, resource type, and log name."
}

func (t *SearchLogsTool) Schema() types.Schema {
	return types.Schema{
		Type: "object",
		Properties: map[string]types.Schema{
			"query": {
				Type: "string",
			},
			"startTime": {
				Type: "string",
			},
			"endTime": {
				Type: "string",
			},
			"severity": {
				Type: "string",
			},
			"resource": {
				Type: "string",
			},
			"logName": {
				Type: "string",
			},
			"pageSize": {
				Type: "integer",
			},
			"orderBy": {
				Type: "string",
			},
		},
		Required: []string{"query"},
		AdditionalProperties: false,
	}
}

func (t *SearchLogsTool) Execute(args map[string]interface{}) (*types.CallToolResult, error) {
	var params SearchLogsArgs
	if argsBytes, err := json.Marshal(args); err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	} else if err := json.Unmarshal(argsBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	if params.Query == "" {
		return &types.CallToolResult{
			Content: []types.Content{{
				Type: "text",
				Text: "Error: query parameter is required",
			}},
			IsError: true,
		}, nil
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
		log.Printf("Cache hit for query: %s", params.Query)
		result := map[string]interface{}{
			"query":   params.Query,
			"count":   len(cachedEntries),
			"entries": cachedEntries,
			"cached":  true,
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

	ctx := context.Background()
	var entries []LogEntry
	var err error
	
	// Execute with rate limiting and backoff
	err = t.rateLimiter.ExecuteWithBackoff(ctx, func() error {
		entries, err = t.searchLogs(ctx, params)
		return err
	})
	if err != nil {
		log.Printf("Failed to search logs: %v", err)
		return &types.CallToolResult{
			Content: []types.Content{{
				Type: "text",
				Text: fmt.Sprintf("Error searching logs: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Cache the results (TTL: 2 minutes for recent logs)
	ttl := 2 * time.Minute
	if params.StartTime != "" {
		// Longer TTL for historical data
		ttl = 10 * time.Minute
	}
	t.cache.Set(cacheKey, entries, ttl)

	result := map[string]interface{}{
		"query":   params.Query,
		"count":   len(entries),
		"entries": entries,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search results: %w", err)
	}

	return &types.CallToolResult{
		Content: []types.Content{{
			Type: "text",
			Text: string(resultJSON),
		}},
	}, nil
}

func (t *SearchLogsTool) searchLogs(ctx context.Context, params SearchLogsArgs) ([]LogEntry, error) {
	client := t.client.LogAdminClient()

	// Build optimized filter using FilterBuilder
	filter := t.buildOptimizedFilter(params)
	
	// Add timeout to prevent long-running queries
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	iter := client.Entries(ctxWithTimeout,
		logadmin.Filter(filter),
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

		if !t.matchesQuery(entry, params.Query) {
			continue
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

func (t *SearchLogsTool) buildOptimizedFilter(params SearchLogsArgs) string {
	fb := NewFilterBuilder()
	
	// Add time constraints first (most efficient for indexing)
	fb.AddTimeRange(params.StartTime, params.EndTime)
	
	// Add severity filter
	fb.AddSeverity(params.Severity)
	
	// Add resource-specific filters
	if params.Resource != "" {
		fb.filters = append(fb.filters, fmt.Sprintf(`resource.type="%s"`, params.Resource))
	}
	
	// Add log name filter
	fb.AddLogName(params.LogName)
	
	// Parse query for Cloud Run services and keywords
	if params.Query != "" {
		t.parseQueryForOptimizedFilters(params.Query, fb)
	}
	
	// Add default time constraint if none specified
	fb.AddDefaultTimeConstraint()
	
	return fb.Build()
}

func (t *SearchLogsTool) parseQueryForOptimizedFilters(query string, fb *FilterBuilder) {
	query = strings.ToLower(query)
	
	// Extract service names (common pattern: service-name-env)
	if strings.Contains(query, "casone") || strings.Contains(query, "tenant") || strings.Contains(query, "api") {
		// Try to extract full service name
		parts := strings.Fields(query)
		for _, part := range parts {
			if strings.Contains(part, "-") && len(part) > 5 {
				fb.AddCloudRunService(part)
				break
			}
		}
	}
	
	// Add keywords for text search
	fb.AddKeywords(query)
}

// Legacy method - kept for backward compatibility
func (t *SearchLogsTool) buildFilter(params SearchLogsArgs) string {
	return t.buildOptimizedFilter(params)
}

func (t *SearchLogsTool) matchesQuery(entry *logging.Entry, query string) bool {
	// Skip client-side filtering if query was converted to server-side filters
	if t.isStructuredQuery(query) {
		return true
	}
	
	query = strings.ToLower(query)

	if strings.Contains(strings.ToLower(entry.LogName), query) {
		return true
	}

	switch payload := entry.Payload.(type) {
	case string:
		if strings.Contains(strings.ToLower(payload), query) {
			return true
		}
	case map[string]interface{}:
		payloadStr, _ := json.Marshal(payload)
		if strings.Contains(strings.ToLower(string(payloadStr)), query) {
			return true
		}
	}

	if entry.Labels != nil {
		for k, v := range entry.Labels {
			if strings.Contains(strings.ToLower(k), query) || strings.Contains(strings.ToLower(v), query) {
				return true
			}
		}
	}

	return false
}

func (t *SearchLogsTool) isStructuredQuery(query string) bool {
	return strings.Contains(query, "service_name") || strings.Contains(query, "cloud_run")
}