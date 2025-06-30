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
	client *Client
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
		client: client,
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
		params.PageSize = 50
	}

	if params.OrderBy == "" {
		params.OrderBy = "timestamp desc"
	}

	ctx := context.Background()
	entries, err := t.searchLogs(ctx, params)
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

	filter := t.buildFilter(params)
	
	iter := client.Entries(ctx,
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

func (t *SearchLogsTool) buildFilter(params SearchLogsArgs) string {
	var filters []string

	if params.StartTime != "" {
		filters = append(filters, fmt.Sprintf(`timestamp >= "%s"`, params.StartTime))
	}

	if params.EndTime != "" {
		filters = append(filters, fmt.Sprintf(`timestamp <= "%s"`, params.EndTime))
	}

	if params.Severity != "" {
		filters = append(filters, fmt.Sprintf(`severity >= "%s"`, strings.ToUpper(params.Severity)))
	}

	if params.Resource != "" {
		filters = append(filters, fmt.Sprintf(`resource.type = "%s"`, params.Resource))
	}

	if params.LogName != "" {
		filters = append(filters, fmt.Sprintf(`logName = "%s"`, params.LogName))
	}

	if len(filters) == 0 {
		return ""
	}

	return strings.Join(filters, " AND ")
}

func (t *SearchLogsTool) matchesQuery(entry *logging.Entry, query string) bool {
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