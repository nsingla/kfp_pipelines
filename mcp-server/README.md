# Kubeflow Pipelines MCP Server

An MCP (Model Context Protocol) server that provides tools for interacting with Kubeflow Pipelines v2beta1 API.

## Features

- **List Pipeline Versions**: List all versions of a specific pipeline
- **Upload Pipeline**: Upload a new pipeline to Kubeflow Pipelines

## Installation

1. Clone the repository
2. Navigate to the mcp-server directory
3. Install dependencies:
   ```bash
   go mod tidy
   ```

## Configuration

Set the following environment variable:
- `KFP_HOST`: The host and port of your Kubeflow Pipelines API server (default: `localhost:8888`)

## Usage

### Running the Server

```bash
go run main.go
```

The server communicates via JSON-RPC 2.0 over stdin/stdout.

### Available Tools

#### 1. list_pipeline_versions

Lists all versions of a specific pipeline.

**Parameters:**
- `pipeline_id` (required): The ID of the pipeline to list versions for
- `page_size` (optional): Number of versions to return per page (default: 10)
- `page_token` (optional): Token for pagination
- `sort_by` (optional): Field to sort by

**Example:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "list_pipeline_versions",
    "arguments": {
      "pipeline_id": "12345678-1234-1234-1234-123456789012",
      "page_size": 5
    }
  }
}
```

#### 2. upload_pipeline

Uploads a new pipeline to Kubeflow Pipelines.

**Parameters:**
- `name` (required): Name of the pipeline
- `pipeline_url` (required): Local file path to the pipeline package (YAML or zip file)
- `description` (optional): Description of the pipeline
- `namespace` (optional): Namespace for the pipeline (default: "kubeflow")

**Example:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "upload_pipeline",
    "arguments": {
      "name": "My New Pipeline",
      "pipeline_url": "/path/to/pipeline.yaml",
      "description": "A sample pipeline for demonstration",
      "namespace": "kubeflow"
    }
  }
}
```

## Error Handling

The server provides comprehensive error handling for:
- Invalid JSON-RPC requests
- Missing required parameters
- Invalid parameter types
- API communication errors
- File access errors

All errors are returned in standard JSON-RPC 2.0 error format.

## Development

To build the server:
```bash
go build -o mcp-pipeline-server main.go
```

To run tests:
```bash
go test ./...
```