# gco-o11y-mcp
MCP server for Google Cloud Observability

This is a Model Context Protocol (MCP) server that provides access to Google Cloud Logging services through stdio communication.

## Overview
The server communicates via stdio (stdin/stdout) and allows AI assistants to read and search log entries from Google Cloud Logging.

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

## Installation for Claude Code

### 1. Clone the repository
```bash
git clone https://github.com/takashabe/gco-o11y-mcp.git
cd gco-o11y-mcp
```

### 2. Set up Google Cloud authentication
```bash
gcloud auth application-default login
```

### 3. Configure Claude Code
Add the following to your Claude Code configuration:

Option 1: Using `.mcp.json` in the project directory
```json
{
  "name": "gcp-o11y",
  "description": "Google Cloud Observability MCP Server",
  "command": "go",
  "args": ["run", "cmd/server/main.go"],
  "env": {
    "GOOGLE_CLOUD_PROJECT": "your-project-id"
  }
}
```

Option 2: Build and use the binary
```bash
# Build the server
task build

# Update .mcp.json
{
  "name": "gcp-o11y",
  "description": "Google Cloud Observability MCP Server",
  "command": "/path/to/gco-o11y-mcp/server",
  "env": {
    "GOOGLE_CLOUD_PROJECT": "your-project-id"
  }
}
```

### 4. Restart Claude Code
After configuration, restart Claude Code to load the MCP server.

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

## Usage in Claude Code

Once the MCP server is configured and Claude Code is restarted, you can use natural language to interact with Google Cloud Logging:

- "List the most recent error logs"
- "Search for logs containing 'timeout' in the last hour"
- "Show me all ERROR severity logs from today"
- "Find logs with 'authentication failed' message"

## Testing the Server Manually

### Test stdio communication
```bash
# Start the server
export GOOGLE_CLOUD_PROJECT=your-project-id
go run cmd/server/main.go

# Send initialize request
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}

# List available tools
{"jsonrpc":"2.0","id":2,"method":"tools/list"}

# Call a tool
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_log_entries","arguments":{"pageSize":5}}}
```

## Authentication
The server uses Google Cloud default credentials. In Cloud Run, this is handled automatically via the service account. For local development, use:

```bash
gcloud auth application-default login
```

## License
This project is licensed under the MIT License - see the LICENSE file for details.
