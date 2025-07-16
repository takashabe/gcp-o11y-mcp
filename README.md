# GCP Observability MCP Server

Model Context Protocol (MCP) server for Google Cloud Platform Observability

## Overview

This project provides an MCP server that offers access to Google Cloud Logging services. It enables AI assistants to read and search log entries from Google Cloud Logging through stdio communication.

## Features

### Available Tools
- **list_log_entries**: List log entries with optional filtering capabilities
- **search_logs**: Advanced log search using text queries and filters
- **preset_query**: Efficient log search with predefined optimized queries

### Performance Optimizations
- **Quota optimization**: Reduced API usage through page size limits and caching
- **Rate limiting**: Automatic retry with exponential backoff
- **In-memory cache**: Prevents duplicate queries (2-10 minute cache duration)
- **Efficient filtering**: Server-side filtering reduces data transfer

## Prerequisites
- Go 1.24.4+
- Google Cloud Project with Logging API enabled
- Service Account with appropriate permissions:
  - `logging.entries.list`
  - `logging.logEntries.list`

## Environment Variables
- `GOOGLE_CLOUD_PROJECT`: Your Google Cloud Project ID (required)

## Installation

### 1. Clone the repository
```bash
git clone https://github.com/takashabe/gcp-o11y-mcp.git
cd gcp-o11y-mcp
```

### 2. Set up Google Cloud authentication
```bash
gcloud auth application-default login
```

### 3. Install dependencies
```bash
task tidy
```

### 4. MCP Configuration
Create `.mcp.json` in the project directory:

```json
{
  "name": "gcp-o11y",
  "description": "Google Cloud Observability MCP Server",
  "command": "go",
  "args": ["run", "."],
  "env": {
    "GOOGLE_CLOUD_PROJECT": "your-project-id"
  }
}
```

### 5. Restart Claude Code
After configuration, restart Claude Code to load the MCP server.

## Usage

### Using with Claude Code
Once the MCP server is configured and Claude Code is restarted, you can interact with Google Cloud Logging using natural language:

- "Show me the latest error logs"
- "Search for logs containing 'timeout' in the last hour"
- "Display all ERROR severity logs from today"
- "Find logs with authentication failure messages"

### Preset Queries
For efficient searching, the following preset queries are available:

- `cloud_run_errors`: Recent errors from Cloud Run services
- `cloud_run_service_errors`: Errors for specific Cloud Run service
- `recent_logs`: Logs from the last hour
- `high_severity`: Critical and error logs from the last 6 hours

## Development & Testing

### Available Tasks
```bash
# List available tasks
task

# Run tests
task test

# Format code
task fmt

# Tidy dependencies
task tidy
```

### Manual Testing
```bash
# Start server
export GOOGLE_CLOUD_PROJECT=your-project-id
task run

# Test JSON-RPC communication
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | task run
```

## Cloud Run Deployment

### Deploy using configuration file
```bash
# Update PROJECT_ID in cloudrun.yaml
sed -i 's/PROJECT_ID/your-actual-project-id/g' cloudrun.yaml

# Deploy
gcloud run services replace cloudrun.yaml --region us-central1
```

## Authentication
The server uses Google Cloud default credentials. For local development, run:

```bash
gcloud auth application-default login
```

## Project Structure
```
.
├── internal/
│   ├── logging/          # Log processing logic
│   │   ├── client.go     # Google Cloud Logging client
│   │   ├── cache.go      # In-memory cache
│   │   ├── ratelimit.go  # Rate limiting
│   │   └── *.go          # Tool implementations
│   └── mcp/              # MCP protocol implementation
├── pkg/types/            # Type definitions
├── Taskfile.yml         # Task definitions
└── cloudrun.yaml        # Cloud Run configuration
```

## License
This project is licensed under the MIT License - see the LICENSE file for details.
