# Kubeflow Pipelines AI Agent

An intelligent AI Agent that can automatically analyze pipeline files, compare with existing versions, generate unique pipelines using different component types, and upload them to Kubeflow Pipelines via the MCP server.

## Features

### Core Capabilities
1. **Pipeline File Analysis**: Reads and analyzes YAML pipeline files from `../data/pipeline_files/valid`
2. **Version Comparison**: Compares with existing pipeline versions using MCP server's `list_pipeline_versions` tool
3. **SDK Component Analysis**: Analyzes different types of pipeline component configurations from the SDK
4. **Unique Pipeline Generation**: Compiles unique pipeline YAML files with 1 or more components
5. **Automated Upload**: Uses MCP server's `upload_pipeline` tool to upload compiled pipelines
6. **REST API**: Provides "Upload Pipeline" API endpoint for workflow automation

### Component Types Supported
- **Data Processor**: Processes and transforms input data
- **Model Trainer**: Trains machine learning models  
- **Data Validator**: Validates data quality and schema
- **Metric Calculator**: Calculates performance metrics
- **Data Loader**: Loads data from various sources

## Architecture

```
AI Agent (Port 8080) <-> MCP Server <-> Kubeflow Pipelines API
```

The AI Agent:
- Starts and manages the MCP server as a subprocess
- Communicates with MCP server via JSON-RPC 2.0 protocol
- Provides a web interface and REST API for user interaction
- Automatically generates unique pipelines with random component combinations

## Installation & Usage

### Prerequisites
- Go 1.24.2 or later
- Kubeflow Pipelines server running (or accessible MCP server endpoint)

### Setup
1. Navigate to the ai-agent directory:
   ```bash
   cd ai-agent
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

### Running the AI Agent

1. Start the AI Agent server:
   ```bash
   go run main.go
   ```

2. The server will start on port 8080 and automatically launch the MCP server as a subprocess.

3. Access the web interface at: `http://localhost:8080`

### API Endpoints

#### POST /api/v1/upload-pipeline
Executes the complete upload workflow:
- Reads pipeline files from `./data/pipeline_files/valid`
- Analyzes SDK component types
- Generates a unique pipeline
- Uploads it via MCP server

**Response:**
```json
{
  "message": "Pipeline uploaded successfully",
  "result": {
    "start_time": "2024-01-01T12:00:00Z",
    "end_time": "2024-01-01T12:00:05Z",
    "duration": "5s",
    "success": true,
    "steps": [...],
    "pipeline_name": "ai-generated-pipeline-20240101-120000-abc12345",
    "pipeline_file": "./generated_pipelines/ai-generated-pipeline-20240101-120000-abc12345.yaml",
    "yaml_content": "# PIPELINE DEFINITION..."
  }
}
```

#### GET /api/v1/component-types
Lists all available component types and their configurations.

#### GET /api/v1/health
Returns health status of the AI Agent.

## Workflow Details

The Upload Pipeline workflow executes these steps:

1. **Read Pipeline Files**: Scans `../data/pipeline_files/valid` for YAML files
2. **Analyze Structures**: Parses sample pipelines to understand format
3. **SDK Analysis**: Identifies available component types from SDK structure  
4. **Version Check**: Queries existing pipeline versions via MCP server
5. **Generate Pipeline**: Creates unique pipeline with 2-4 random components
6. **Compile YAML**: Converts pipeline spec to valid Kubeflow Pipelines YAML
7. **Save File**: Writes generated pipeline to `./generated_pipelines/`
8. **Upload**: Uploads pipeline via MCP server to Kubeflow Pipelines

## Generated Pipeline Structure

Each generated pipeline includes:
- Unique name with timestamp and UUID
- Descriptive metadata
- 2-4 components randomly selected from available types
- Proper component dependencies and data flow
- Kubernetes executor specifications
- Caching and retry configurations

## Configuration

### Environment Variables
- `KFP_HOST`: Kubeflow Pipelines host (default: localhost:8888)

### Directory Structure
```
ai-agent/
├── main.go                     # Main AI Agent implementation
├── go.mod                      # Go module dependencies
├── README.md                   # This file
├── generated_pipelines/        # Output directory for generated pipelines
└── ../data/pipeline_files/valid/  # Input pipeline files (relative path)
```

## Testing

### Manual Testing
1. Start the server: `go run main.go`
2. Visit `http://localhost:8080`
3. Click "Upload Pipeline" to test the complete workflow
4. Check the generated pipeline in `./generated_pipelines/`

### API Testing
```bash
# Test upload workflow
curl -X POST http://localhost:8080/api/v1/upload-pipeline

# List component types  
curl http://localhost:8080/api/v1/component-types

# Health check
curl http://localhost:8080/api/v1/health
```

## Error Handling

The AI Agent provides comprehensive error handling:
- MCP server communication failures
- Pipeline file parsing errors
- YAML compilation issues
- Upload failures
- File system errors

All errors are logged and returned in structured JSON responses with detailed workflow steps.

## Development

### Adding New Component Types
Edit the `componentTypes` array in `NewAIAgent()` to add new component configurations:

```go
{
    Type:        "new_component_type",
    Name:        "New Component",
    Description: "Description of the new component",
    Image:       "container-image:tag",
    Command:     []string{"command"},
    Args:        []string{"arg1", "arg2"},
    Inputs:      map[string]string{"input1": "STRING"},
    Outputs:     map[string]string{"output1": "STRING"},
},
```

### Extending Pipeline Generation
Modify the `GenerateUniquePipeline()` method to customize:
- Component selection logic
- Pipeline naming conventions
- Component interconnections
- Resource specifications