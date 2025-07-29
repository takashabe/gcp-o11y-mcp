package logging

import (
	"fmt"
	"time"
)

type PresetQuery struct {
	Name        string
	Description string
	Filter      string
	PageSize    int
}

var CommonPresetQueries = map[string]PresetQuery{
	"cloud_run_errors": {
		Name:        "cloud_run_errors",
		Description: "Get recent errors from Cloud Run services",
		Filter:      `resource.type="cloud_run_revision" AND severity>=ERROR AND timestamp>="%s"`,
		PageSize:    10,
	},
	"cloud_run_service_errors": {
		Name:        "cloud_run_service_errors",
		Description: "Get errors for specific Cloud Run service",
		Filter:      `resource.type="cloud_run_revision" AND resource.labels.service_name="%s" AND severity>=ERROR AND timestamp>="%s"`,
		PageSize:    15,
	},
	"recent_logs": {
		Name:        "recent_logs",
		Description: "Get recent logs from last hour",
		Filter:      `timestamp>="%s"`,
		PageSize:    20,
	},
	"high_severity": {
		Name:        "high_severity",
		Description: "Get critical and error logs from last 6 hours",
		Filter:      `severity>=ERROR AND timestamp>="%s"`,
		PageSize:    10,
	},
}

func GetPresetQuery(queryName string, params ...string) (string, int, error) {
	preset, exists := CommonPresetQueries[queryName]
	if !exists {
		return "", 0, fmt.Errorf("preset query '%s' not found", queryName)
	}

	switch queryName {
	case "cloud_run_errors":
		defaultTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		filter := fmt.Sprintf(preset.Filter, defaultTime)
		return filter, preset.PageSize, nil

	case "cloud_run_service_errors":
		if len(params) < 1 {
			return "", 0, fmt.Errorf("service name parameter required for cloud_run_service_errors")
		}
		serviceName := params[0]
		defaultTime := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
		filter := fmt.Sprintf(preset.Filter, serviceName, defaultTime)
		return filter, preset.PageSize, nil

	case "recent_logs":
		defaultTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		filter := fmt.Sprintf(preset.Filter, defaultTime)
		return filter, preset.PageSize, nil

	case "high_severity":
		defaultTime := time.Now().Add(-6 * time.Hour).Format(time.RFC3339)
		filter := fmt.Sprintf(preset.Filter, defaultTime)
		return filter, preset.PageSize, nil

	default:
		return preset.Filter, preset.PageSize, nil
	}
}

func ListPresetQueries() []PresetQuery {
	queries := make([]PresetQuery, 0, len(CommonPresetQueries))
	for _, query := range CommonPresetQueries {
		queries = append(queries, query)
	}
	return queries
}
