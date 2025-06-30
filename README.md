# gco-o11y-mcp
MCP server for Google Cloud Observability

## Overview
This is a Model Context Protocol (MCP) server that provides access to Google Cloud Logging services. It allows AI assistants to read and search log entries from Google Cloud Logging.

## Features
- **list_log_entries**: List log entries with optional filtering
- **search_logs**: Advanced log search with text queries and filters

## Prerequisites
- Go 1.24+
- Google Cloud Project with Logging API enabled
- Service Account with appropriate permissions:
  - `logging.entries.list`
  - `logging.logEntries.list`

## Environment Variables
- `GOOGLE_CLOUD_PROJECT`: Your Google Cloud Project ID (required)
- `PORT`: Server port (default: 8080)

## Local Development

### Setup
```bash
go mod download
```

### Run locally
```bash
export GOOGLE_CLOUD_PROJECT=your-project-id
go run cmd/server/main.go
```

### Test the server
```bash
curl -X POST http://localhost:8080 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

## Cloud Run Deployment

### Build and deploy
```bash
# Build and push the container
gcloud builds submit --tag gcr.io/PROJECT_ID/gco-o11y-mcp

# Deploy to Cloud Run
gcloud run deploy gco-o11y-mcp \
  --image gcr.io/PROJECT_ID/gco-o11y-mcp \
  --platform managed \
  --region us-central1 \
  --set-env-vars GOOGLE_CLOUD_PROJECT=PROJECT_ID \
  --allow-unauthenticated
```

### Using the config file
```bash
# Update PROJECT_ID in cloudrun.yaml
sed -i 's/PROJECT_ID/your-actual-project-id/g' cloudrun.yaml

# Deploy using config
gcloud run services replace cloudrun.yaml --region us-central1
```

## Usage Examples

### List recent log entries
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "list_log_entries",
    "arguments": {
      "pageSize": 10,
      "orderBy": "timestamp desc"
    }
  }
}
```

### Search logs with filter
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "search_logs",
    "arguments": {
      "query": "error",
      "severity": "ERROR",
      "startTime": "2024-01-01T00:00:00Z",
      "pageSize": 20
    }
  }
}
```

## Authentication
The server uses Google Cloud default credentials. In Cloud Run, this is handled automatically via the service account. For local development, use:

```bash
gcloud auth application-default login
```

## License
This project is licensed under the MIT License - see the LICENSE file for details.
