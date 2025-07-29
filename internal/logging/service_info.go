package logging

import (
	"encoding/json"
	"time"
)

type ServiceInfo struct {
	ServiceName  string                 `json:"service_name"`
	ProjectID    string                 `json:"project_id"`
	Location     string                 `json:"location"`
	RevisionName string                 `json:"revision_name"`
	InstanceID   string                 `json:"instance_id"`
	Timestamp    time.Time              `json:"timestamp"`
	Severity     string                 `json:"severity"`
	Message      string                 `json:"message"`
	Labels       map[string]string      `json:"labels,omitempty"`
	JSONPayload  map[string]interface{} `json:"json_payload,omitempty"`
	TraceID      string                 `json:"trace_id,omitempty"`
	SpanID       string                 `json:"span_id,omitempty"`
}

func ExtractServiceInfoFromLogEntry(entry LogEntry) *ServiceInfo {
	info := &ServiceInfo{
		Severity: entry.Severity,
		TraceID:  entry.TraceID,
		Labels:   entry.Labels,
	}

	// Parse timestamp
	if timestamp, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
		info.Timestamp = timestamp
	}

	// Extract resource information
	if entry.Resource != nil {
		if resourceType, ok := entry.Resource["type"].(string); ok && resourceType == "cloud_run_revision" {
			if labels, ok := entry.Resource["labels"].(map[string]interface{}); ok {
				if serviceName, ok := labels["service_name"].(string); ok {
					info.ServiceName = serviceName
				}
				if projectID, ok := labels["project_id"].(string); ok {
					info.ProjectID = projectID
				}
				if location, ok := labels["location"].(string); ok {
					info.Location = location
				}
				if revisionName, ok := labels["revision_name"].(string); ok {
					info.RevisionName = revisionName
				}
			}
		}
	}

	// Extract message from payload
	if entry.TextPayload != "" {
		info.Message = entry.TextPayload
	} else if entry.JSONPayload != nil {
		info.JSONPayload = entry.JSONPayload

		// Try to extract message from common JSON fields
		if message, ok := entry.JSONPayload["message"].(string); ok {
			info.Message = message
		} else if msg, ok := entry.JSONPayload["msg"].(string); ok {
			info.Message = msg
		} else {
			// Convert JSON payload to string as fallback
			if jsonBytes, err := json.Marshal(entry.JSONPayload); err == nil {
				info.Message = string(jsonBytes)
			}
		}
	}

	return info
}

func (si *ServiceInfo) ToLogEntry() LogEntry {
	entry := LogEntry{
		Timestamp:   si.Timestamp.Format(time.RFC3339),
		Severity:    si.Severity,
		TextPayload: si.Message,
		Labels:      si.Labels,
		TraceID:     si.TraceID,
	}

	if si.JSONPayload != nil {
		entry.JSONPayload = si.JSONPayload
		entry.TextPayload = ""
	}

	entry.Resource = map[string]interface{}{
		"type": "cloud_run_revision",
		"labels": map[string]interface{}{
			"service_name":  si.ServiceName,
			"project_id":    si.ProjectID,
			"location":      si.Location,
			"revision_name": si.RevisionName,
		},
	}

	return entry
}
