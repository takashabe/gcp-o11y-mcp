package logging

import (
	"fmt"
	"strings"
	"time"
)

type FilterBuilder struct {
	filters []string
}

func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{
		filters: make([]string, 0),
	}
}

func (fb *FilterBuilder) AddTimeRange(startTime, endTime string) *FilterBuilder {
	if startTime != "" {
		fb.filters = append(fb.filters, fmt.Sprintf(`timestamp >= "%s"`, startTime))
	}
	if endTime != "" {
		fb.filters = append(fb.filters, fmt.Sprintf(`timestamp <= "%s"`, endTime))
	}
	return fb
}

func (fb *FilterBuilder) AddSeverity(severity string) *FilterBuilder {
	if severity != "" {
		fb.filters = append(fb.filters, fmt.Sprintf(`severity >= %s`, strings.ToUpper(severity)))
	}
	return fb
}

func (fb *FilterBuilder) AddCloudRunService(serviceName string) *FilterBuilder {
	if serviceName != "" {
		fb.filters = append(fb.filters, `resource.type="cloud_run_revision"`)
		fb.filters = append(fb.filters, fmt.Sprintf(`resource.labels.service_name="%s"`, serviceName))
	}
	return fb
}

func (fb *FilterBuilder) AddLogName(logName string) *FilterBuilder {
	if logName != "" {
		fb.filters = append(fb.filters, fmt.Sprintf(`logName="%s"`, logName))
	}
	return fb
}

func (fb *FilterBuilder) AddKeywords(keywords string) *FilterBuilder {
	if keywords != "" {
		// Try to identify if keywords contain structured information
		if strings.Contains(keywords, "error") || strings.Contains(keywords, "ERROR") {
			fb.filters = append(fb.filters, `severity >= ERROR`)
		}
		
		// For text search, use textPayload or jsonPayload
		keywordFilter := fmt.Sprintf(`(textPayload:"%s" OR jsonPayload.message:"%s")`, keywords, keywords)
		fb.filters = append(fb.filters, keywordFilter)
	}
	return fb
}

func (fb *FilterBuilder) AddDefaultTimeConstraint() *FilterBuilder {
	// Add default 24-hour constraint if no time filters exist
	hasTimeFilter := false
	for _, filter := range fb.filters {
		if strings.Contains(filter, "timestamp") {
			hasTimeFilter = true
			break
		}
	}
	
	if !hasTimeFilter {
		defaultStart := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
		fb.filters = append(fb.filters, fmt.Sprintf(`timestamp >= "%s"`, defaultStart))
	}
	return fb
}

func (fb *FilterBuilder) Build() string {
	if len(fb.filters) == 0 {
		return ""
	}
	return strings.Join(fb.filters, " AND ")
}

func (fb *FilterBuilder) Reset() *FilterBuilder {
	fb.filters = make([]string, 0)
	return fb
}