package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
)

type MCPRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type MCPResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeParams struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      map[string]interface{} `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      map[string]interface{} `json:"serverInfo"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []ToolContent `json:"content"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type PipelineServer struct {
	pipelineClient       *pipeline_client.Pipeline
	pipelineUploadClient *pipeline_upload_client.PipelineUpload
	host                 string
}

func NewPipelineServer(host string) *PipelineServer {
	transport := httptransport.New(host, "/api/v2beta1", []string{"http"})
	pipelineClient := pipeline_client.New(transport, strfmt.Default)
	pipelineUploadClient := pipeline_upload_client.New(transport, strfmt.Default)

	return &PipelineServer{
		pipelineClient:       pipelineClient,
		pipelineUploadClient: pipelineUploadClient,
		host:                 host,
	}
}

func main() {
	host := os.Getenv("KFP_HOST")
	if host == "" {
		host = "localhost:8888"
	}

	server := NewPipelineServer(host)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req MCPRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			log.Printf("Error parsing request: %v", err)
			continue
		}

		response := server.handleRequest(req)
		
		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Printf("Error marshaling response: %v", err)
			continue
		}

		fmt.Println(string(responseJSON))
	}
}

func (s *PipelineServer) handleRequest(req MCPRequest) MCPResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleListTools(req)
	case "tools/call":
		return s.handleCallTool(req)
	default:
		return MCPResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

func (s *PipelineServer) handleInitialize(req MCPRequest) MCPResponse {
	return MCPResponse{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Result: InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			ServerInfo: map[string]interface{}{
				"name":    "kubeflow-pipelines-mcp",
				"version": "1.0.0",
			},
		},
	}
}

func (s *PipelineServer) handleListTools(req MCPRequest) MCPResponse {
	tools := []Tool{
		{
			Name:        "list_pipeline_versions",
			Description: "List all versions of a specific pipeline",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pipeline_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the pipeline to list versions for",
					},
					"page_size": map[string]interface{}{
						"type":        "integer",
						"description": "Number of versions to return per page (optional, default: 10)",
						"default":     10,
					},
					"page_token": map[string]interface{}{
						"type":        "string",
						"description": "Token for pagination (optional)",
					},
					"sort_by": map[string]interface{}{
						"type":        "string",
						"description": "Field to sort by (optional)",
					},
				},
				"required": []string{"pipeline_id"},
			},
		},
		{
			Name:        "upload_pipeline",
			Description: "Upload a new pipeline to Kubeflow Pipelines",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the pipeline",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Description of the pipeline (optional)",
					},
					"pipeline_url": map[string]interface{}{
						"type":        "string",
						"description": "URL to the pipeline package (YAML or zip file)",
					},
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Namespace for the pipeline (optional, defaults to 'kubeflow')",
						"default":     "kubeflow",
					},
				},
				"required": []string{"name", "pipeline_url"},
			},
		},
	}

	return MCPResponse{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Result:  ListToolsResult{Tools: tools},
	}
}

func (s *PipelineServer) handleCallTool(req MCPRequest) MCPResponse {
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return MCPResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}

	var callParams CallToolParams
	if err := json.Unmarshal(paramsBytes, &callParams); err != nil {
		return MCPResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params format",
			},
		}
	}

	switch callParams.Name {
	case "list_pipeline_versions":
		return s.handleListPipelineVersions(req.ID, callParams.Arguments)
	case "upload_pipeline":
		return s.handleUploadPipeline(req.ID, callParams.Arguments)
	default:
		return MCPResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Tool not found: %s", callParams.Name),
			},
		}
	}
}

func (s *PipelineServer) handleListPipelineVersions(id interface{}, args map[string]interface{}) MCPResponse {
	pipelineID, ok := args["pipeline_id"].(string)
	if !ok || pipelineID == "" {
		return MCPResponse{
			Jsonrpc: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "pipeline_id is required and must be a non-empty string",
			},
		}
	}

	pageSize := int32(10)
	if ps, ok := args["page_size"]; ok {
		if psFloat, ok := ps.(float64); ok {
			if psFloat <= 0 || psFloat > 100 {
				return MCPResponse{
					Jsonrpc: "2.0",
					ID:      id,
					Error: &MCPError{
						Code:    -32602,
						Message: "page_size must be between 1 and 100",
					},
				}
			}
			pageSize = int32(psFloat)
		} else {
			return MCPResponse{
				Jsonrpc: "2.0",
				ID:      id,
				Error: &MCPError{
					Code:    -32602,
					Message: "page_size must be a number",
				},
			}
		}
	}

	var pageToken *string
	if pt, ok := args["page_token"].(string); ok && pt != "" {
		pageToken = &pt
	}

	var sortBy *string
	if sb, ok := args["sort_by"].(string); ok && sb != "" {
		sortBy = &sb
	}

	params := pipeline_service.NewPipelineServiceListPipelineVersionsParams()
	params.PipelineID = pipelineID
	params.PageSize = &pageSize
	if pageToken != nil {
		params.PageToken = pageToken
	}
	if sortBy != nil {
		params.SortBy = sortBy
	}

	resp, err := s.pipelineClient.PipelineService.PipelineServiceListPipelineVersions(params, nil)
	if err != nil {
		return MCPResponse{
			Jsonrpc: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32603,
				Message: fmt.Sprintf("Failed to list pipeline versions: %v", err),
			},
		}
	}

	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Pipeline Versions for Pipeline ID: %s\n\n", pipelineID))

	if resp.Payload.PipelineVersions != nil && len(resp.Payload.PipelineVersions) > 0 {
		for i, version := range resp.Payload.PipelineVersions {
			resultText.WriteString(fmt.Sprintf("%d. Name: %s\n", i+1, version.Name))
			resultText.WriteString(fmt.Sprintf("   ID: %s\n", version.PipelineVersionID))
			if version.Description != "" {
				resultText.WriteString(fmt.Sprintf("   Description: %s\n", version.Description))
			}
			if version.CreatedAt.String() != "" {
				resultText.WriteString(fmt.Sprintf("   Created: %s\n", version.CreatedAt.String()))
			}
			if version.CodeSourceURL != "" {
				resultText.WriteString(fmt.Sprintf("   Code Source: %s\n", version.CodeSourceURL))
			}
			resultText.WriteString("\n")
		}

		if resp.Payload.NextPageToken != "" {
			resultText.WriteString(fmt.Sprintf("Next page token: %s\n", resp.Payload.NextPageToken))
		}

		if resp.Payload.TotalSize != 0 {
			resultText.WriteString(fmt.Sprintf("Total versions: %d\n", resp.Payload.TotalSize))
		}
	} else {
		resultText.WriteString("No pipeline versions found.\n")
	}

	return MCPResponse{
		Jsonrpc: "2.0",
		ID:      id,
		Result: CallToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: resultText.String(),
				},
			},
		},
	}
}

func (s *PipelineServer) handleUploadPipeline(id interface{}, args map[string]interface{}) MCPResponse {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return MCPResponse{
			Jsonrpc: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "name is required and must be a non-empty string",
			},
		}
	}

	pipelineURL, ok := args["pipeline_url"].(string)
	if !ok || pipelineURL == "" {
		return MCPResponse{
			Jsonrpc: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "pipeline_url is required and must be a non-empty string",
			},
		}
	}

	// Validate file exists and is readable
	if _, err := os.Stat(pipelineURL); os.IsNotExist(err) {
		return MCPResponse{
			Jsonrpc: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: fmt.Sprintf("pipeline file does not exist: %s", pipelineURL),
			},
		}
	}

	description := ""
	if desc, ok := args["description"].(string); ok {
		description = desc
	}

	namespace := "kubeflow"
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	params := pipeline_upload_service.NewUploadPipelineParams()
	
	uploadfile, err := os.Open(pipelineURL)
	if err != nil {
		return MCPResponse{
			Jsonrpc: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32603,
				Message: fmt.Sprintf("Failed to open pipeline file: %v", err),
			},
		}
	}
	defer uploadfile.Close()

	params.Uploadfile = runtime.NamedReader("uploadfile", uploadfile)
	params.Name = &name
	if description != "" {
		params.Description = &description
	}
	params.Namespace = &namespace

	resp, err := s.pipelineUploadClient.PipelineUploadService.UploadPipeline(params, nil)
	if err != nil {
		return MCPResponse{
			Jsonrpc: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32603,
				Message: fmt.Sprintf("Failed to upload pipeline: %v", err),
			},
		}
	}

	var resultText strings.Builder
	resultText.WriteString("Pipeline uploaded successfully!\n\n")
	
	if resp.Payload != nil {
		resultText.WriteString(fmt.Sprintf("Pipeline ID: %s\n", resp.Payload.PipelineID))
		resultText.WriteString(fmt.Sprintf("Name: %s\n", resp.Payload.Name))
		if resp.Payload.Description != "" {
			resultText.WriteString(fmt.Sprintf("Description: %s\n", resp.Payload.Description))
		}
		if resp.Payload.CreatedAt.String() != "" {
			resultText.WriteString(fmt.Sprintf("Created: %s\n", resp.Payload.CreatedAt.String()))
		}
		resultText.WriteString(fmt.Sprintf("Namespace: %s\n", resp.Payload.Namespace))
	}

	return MCPResponse{
		Jsonrpc: "2.0",
		ID:      id,
		Result: CallToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: resultText.String(),
				},
			},
		},
	}
}