package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
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

type PipelineInfo struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

type ComponentSpec struct {
	ExecutorLabel     string                 `yaml:"executorLabel"`
	InputDefinitions  map[string]interface{} `yaml:"inputDefinitions,omitempty"`
	OutputDefinitions map[string]interface{} `yaml:"outputDefinitions,omitempty"`
}

type ExecutorSpec struct {
	Container map[string]interface{} `yaml:"container"`
}

type TaskSpec struct {
	ComponentRef   map[string]string      `yaml:"componentRef"`
	TaskInfo       map[string]string      `yaml:"taskInfo"`
	CachingOptions map[string]interface{} `yaml:"cachingOptions,omitempty"`
}

type PipelineSpec struct {
	PipelineInfo   PipelineInfo             `yaml:"pipelineInfo"`
	Components     map[string]ComponentSpec `yaml:"components"`
	DeploymentSpec struct {
		Executors map[string]ExecutorSpec `yaml:"executors"`
	} `yaml:"deploymentSpec"`
	Root struct {
		DAG struct {
			Tasks map[string]TaskSpec `yaml:"tasks"`
		} `yaml:"dag"`
	} `yaml:"root"`
	SchemaVersion string `yaml:"schemaVersion"`
	SDKVersion    string `yaml:"sdkVersion"`
}

type ComponentConfig struct {
	Type        string
	Name        string
	Description string
	Image       string
	Command     []string
	Args        []string
	Inputs      map[string]string
	Outputs     map[string]string
}

type AIAgent struct {
	mcpServerCmd    *exec.Cmd
	mcpServerStdin  io.WriteCloser
	mcpServerStdout io.ReadCloser
	componentTypes  []ComponentConfig
}

func NewAIAgent() *AIAgent {
	return &AIAgent{
		componentTypes: []ComponentConfig{
			{
				Type:        "data_processor",
				Name:        "Data Processor",
				Description: "Processes and transforms input data",
				Image:       "python:3.9",
				Command:     []string{"python", "-c"},
				Args:        []string{"import pandas as pd; print('Data processed successfully')"},
				Inputs:      map[string]string{"data": "STRING"},
				Outputs:     map[string]string{"processed_data": "STRING"},
			},
			{
				Type:        "model_trainer",
				Name:        "Model Trainer",
				Description: "Trains machine learning models",
				Image:       "python:3.9",
				Command:     []string{"python", "-c"},
				Args:        []string{"import sklearn; print('Model trained successfully')"},
				Inputs:      map[string]string{"training_data": "STRING", "model_config": "STRING"},
				Outputs:     map[string]string{"model": "MODEL"},
			},
			{
				Type:        "data_validator",
				Name:        "Data Validator",
				Description: "Validates data quality and schema",
				Image:       "python:3.9",
				Command:     []string{"python", "-c"},
				Args:        []string{"print('Data validation completed')"},
				Inputs:      map[string]string{"data": "STRING"},
				Outputs:     map[string]string{"validation_result": "STRING"},
			},
			{
				Type:        "metric_calculator",
				Name:        "Metric Calculator",
				Description: "Calculates performance metrics",
				Image:       "python:3.9",
				Command:     []string{"python", "-c"},
				Args:        []string{"print('Metrics calculated successfully')"},
				Inputs:      map[string]string{"predictions": "STRING", "ground_truth": "STRING"},
				Outputs:     map[string]string{"metrics": "METRICS"},
			},
			{
				Type:        "data_loader",
				Name:        "Data Loader",
				Description: "Loads data from various sources",
				Image:       "python:3.9",
				Command:     []string{"python", "-c"},
				Args:        []string{"print('Data loaded successfully')"},
				Inputs:      map[string]string{"source_path": "STRING"},
				Outputs:     map[string]string{"data": "DATASET"},
			},
		},
	}
}

func (a *AIAgent) StartMCPServer() error {
	cmd := exec.Command("go", "run", "../mcp-server/main.go")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %v", err)
	}

	a.mcpServerCmd = cmd
	a.mcpServerStdin = stdin
	a.mcpServerStdout = stdout

	// Initialize the MCP server
	initRequest := MCPRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "kfp-ai-agent",
				"version": "1.0.0",
			},
		},
	}

	if err := a.sendMCPRequest(initRequest); err != nil {
		return fmt.Errorf("failed to initialize MCP server: %v", err)
	}

	return nil
}

func (a *AIAgent) sendMCPRequest(req MCPRequest) error {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = a.mcpServerStdin.Write(append(reqBytes, '\n'))
	return err
}

func (a *AIAgent) readMCPResponse() (*MCPResponse, error) {
	reader := bufio.NewReader(a.mcpServerStdout)
	line, isPrefix, err := reader.ReadLine()
	if err != nil {
		return nil, err
	}
	if isPrefix {
		return nil, fmt.Errorf("line too long")
	}

	var resp MCPResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (a *AIAgent) callMCPTool(toolName string, args map[string]interface{}) (*CallToolResult, error) {
	req := MCPRequest{
		Jsonrpc: "2.0",
		ID:      time.Now().UnixNano(),
		Method:  "tools/call",
		Params: CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	}

	if err := a.sendMCPRequest(req); err != nil {
		return nil, fmt.Errorf("failed to send MCP request: %v", err)
	}

	resp, err := a.readMCPResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP response: %v", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	var toolResult CallToolResult
	if err := json.Unmarshal(resultBytes, &toolResult); err != nil {
		return nil, err
	}

	return &toolResult, nil
}

func (a *AIAgent) ReadPipelineFiles(directory string) ([]string, error) {
	var files []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func (a *AIAgent) ParsePipelineFile(filePath string) (*PipelineSpec, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var pipeline PipelineSpec
	if err := yaml.Unmarshal(data, &pipeline); err != nil {
		return nil, err
	}

	return &pipeline, nil
}

func (a *AIAgent) ListPipelineVersions(pipelineID string) ([]string, error) {
	result, err := a.callMCPTool("list_pipeline_versions", map[string]interface{}{
		"pipeline_id": pipelineID,
		"page_size":   50,
	})
	if err != nil {
		return nil, err
	}

	if len(result.Content) == 0 {
		return []string{}, nil
	}

	// Parse the text response to extract version information
	lines := strings.Split(result.Content[0].Text, "\n")
	var versions []string

	for _, line := range lines {
		if strings.Contains(line, "Name:") {
			parts := strings.Split(line, "Name:")
			if len(parts) > 1 {
				versions = append(versions, strings.TrimSpace(parts[1]))
			}
		}
	}

	return versions, nil
}

func (a *AIAgent) AnalyzeSDKComponents() []ComponentConfig {
	return a.componentTypes
}

func (a *AIAgent) GenerateUniquePipeline(existingVersions []string) (*PipelineSpec, error) {
	// Generate a unique pipeline name
	timestamp := time.Now().Format("20060102-150405")
	uniqueID := uuid.New().String()[:8]
	pipelineName := fmt.Sprintf("ai-generated-pipeline-%s-%s", timestamp, uniqueID)

	// Ensure uniqueness
	for _, version := range existingVersions {
		if strings.Contains(version, pipelineName) {
			// If somehow we have a conflict, add more randomness
			pipelineName = fmt.Sprintf("%s-%d", pipelineName, rand.Intn(10000))
			break
		}
	}

	// Select random components to include
	numComponents := rand.Intn(3) + 2 // 2-4 components
	selectedComponents := make([]ComponentConfig, 0, numComponents)

	// Always include a data loader as the first component
	selectedComponents = append(selectedComponents, a.componentTypes[4]) // data_loader

	// Add random additional components
	for i := 1; i < numComponents; i++ {
		compIndex := rand.Intn(len(a.componentTypes) - 1) // Exclude data_loader from random selection
		selectedComponents = append(selectedComponents, a.componentTypes[compIndex])
	}

	// Generate pipeline spec
	pipeline := &PipelineSpec{
		PipelineInfo: PipelineInfo{
			Name:        pipelineName,
			Description: fmt.Sprintf("AI-generated pipeline with %d components created at %s", numComponents, time.Now().Format(time.RFC3339)),
		},
		SchemaVersion: "2.1.0",
		SDKVersion:    "kfp-2.13.0",
	}

	pipeline.Components = make(map[string]ComponentSpec)
	pipeline.DeploymentSpec.Executors = make(map[string]ExecutorSpec)
	pipeline.Root.DAG.Tasks = make(map[string]TaskSpec)

	// Generate components, executors, and tasks
	for i, comp := range selectedComponents {
		compName := fmt.Sprintf("comp-%s-%d", comp.Type, i+1)
		execName := fmt.Sprintf("exec-%s-%d", comp.Type, i+1)
		taskName := fmt.Sprintf("%s-%d", comp.Type, i+1)

		// Component specification
		componentSpec := ComponentSpec{
			ExecutorLabel: execName,
		}

		if len(comp.Inputs) > 0 {
			componentSpec.InputDefinitions = map[string]interface{}{
				"parameters": convertInputsToParams(comp.Inputs),
			}
		}

		if len(comp.Outputs) > 0 {
			componentSpec.OutputDefinitions = map[string]interface{}{
				"parameters": convertOutputsToParams(comp.Outputs),
			}
		}

		pipeline.Components[compName] = componentSpec

		// Executor specification
		pipeline.DeploymentSpec.Executors[execName] = ExecutorSpec{
			Container: map[string]interface{}{
				"image":   comp.Image,
				"command": comp.Command,
				"args":    comp.Args,
			},
		}

		// Task specification
		pipeline.Root.DAG.Tasks[taskName] = TaskSpec{
			ComponentRef: map[string]string{
				"name": compName,
			},
			TaskInfo: map[string]string{
				"name": taskName,
			},
			CachingOptions: map[string]interface{}{
				"enableCache": true,
			},
		}
	}

	return pipeline, nil
}

func convertInputsToParams(inputs map[string]string) map[string]interface{} {
	params := make(map[string]interface{})
	for name, paramType := range inputs {
		params[name] = map[string]interface{}{
			"parameterType": convertTypeToParameterType(paramType),
		}
	}
	return params
}

func convertOutputsToParams(outputs map[string]string) map[string]interface{} {
	params := make(map[string]interface{})
	for name, paramType := range outputs {
		params[name] = map[string]interface{}{
			"parameterType": convertTypeToParameterType(paramType),
		}
	}
	return params
}

func convertTypeToParameterType(paramType string) string {
	switch paramType {
	case "STRING":
		return "STRING"
	case "NUMBER":
		return "NUMBER_DOUBLE"
	case "INTEGER":
		return "NUMBER_INTEGER"
	case "MODEL":
		return "STRING" // Models are typically represented as paths/URIs
	case "DATASET":
		return "STRING" // Datasets are typically represented as paths/URIs
	case "METRICS":
		return "STRING" // Metrics are typically JSON strings
	default:
		return "STRING"
	}
}

func (a *AIAgent) CompilePipelineToYAML(pipeline *PipelineSpec) (string, error) {
	yamlData, err := yaml.Marshal(pipeline)
	if err != nil {
		return "", err
	}

	// Add header comment
	header := fmt.Sprintf("# PIPELINE DEFINITION\n# Name: %s\n# Description: %s\n",
		pipeline.PipelineInfo.Name,
		pipeline.PipelineInfo.Description)

	return header + string(yamlData), nil
}

func (a *AIAgent) SavePipelineToFile(yamlContent string, pipelineName string) (string, error) {
	// Create output directory if it doesn't exist
	outputDir := "./generated_pipelines"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	filename := fmt.Sprintf("%s/%s.yaml", outputDir, pipelineName)

	if err := os.WriteFile(filename, []byte(yamlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write pipeline file: %v", err)
	}

	return filename, nil
}

func (a *AIAgent) UploadPipeline(pipelineName, pipelineFile, description string) (*CallToolResult, error) {
	return a.callMCPTool("upload_pipeline", map[string]interface{}{
		"name":         pipelineName,
		"pipeline_url": pipelineFile,
		"description":  description,
		"namespace":    "kubeflow",
	})
}

func (a *AIAgent) ExecuteUploadWorkflow() (*UploadWorkflowResult, error) {
	log.Println("Starting AI Agent Upload Workflow...")

	result := &UploadWorkflowResult{
		StartTime: time.Now(),
		Steps:     []WorkflowStep{},
	}

	// Step 1: Read pipeline files
	result.AddStep("Reading pipeline files from ../data/pipeline_files/valid")
	files, err := a.ReadPipelineFiles("../data/pipeline_files/valid")
	if err != nil {
		result.AddError(fmt.Sprintf("Failed to read pipeline files: %v", err))
		return result, err
	}
	result.AddStep(fmt.Sprintf("Found %d pipeline files", len(files)))

	// Step 2: Parse a sample pipeline to understand structure
	result.AddStep("Analyzing pipeline structures")
	if len(files) > 0 {
		_, err = a.ParsePipelineFile(files[0])
		if err != nil {
			result.AddStep(fmt.Sprintf("Warning: Could not parse sample file %s: %v", files[0], err))
		} else {
			result.AddStep(fmt.Sprintf("Successfully analyzed pipeline structure from %s", filepath.Base(files[0])))
		}
	}

	// Step 3: Analyze SDK components
	result.AddStep("Analyzing SDK component types")
	components := a.AnalyzeSDKComponents()
	result.AddStep(fmt.Sprintf("Found %d component types: %v", len(components), getComponentTypeNames(components)))

	// Step 4: Check existing pipeline versions (use a dummy pipeline ID for demo)
	result.AddStep("Checking existing pipeline versions")
	dummyPipelineID := "sample-pipeline-id-123"
	existingVersions, err := a.ListPipelineVersions(dummyPipelineID)
	if err != nil {
		result.AddStep(fmt.Sprintf("Note: Could not fetch existing versions (expected for demo): %v", err))
		existingVersions = []string{} // Continue with empty list
	} else {
		result.AddStep(fmt.Sprintf("Found %d existing versions", len(existingVersions)))
	}

	// Step 5: Generate unique pipeline
	result.AddStep("Generating unique pipeline")
	newPipeline, err := a.GenerateUniquePipeline(existingVersions)
	if err != nil {
		result.AddError(fmt.Sprintf("Failed to generate pipeline: %v", err))
		return result, err
	}
	result.AddStep(fmt.Sprintf("Generated pipeline: %s", newPipeline.PipelineInfo.Name))

	// Step 6: Compile to YAML
	result.AddStep("Compiling pipeline to YAML")
	yamlContent, err := a.CompilePipelineToYAML(newPipeline)
	if err != nil {
		result.AddError(fmt.Sprintf("Failed to compile pipeline: %v", err))
		return result, err
	}
	result.AddStep("Successfully compiled pipeline to YAML")

	// Step 7: Save to file
	result.AddStep("Saving pipeline to file")
	pipelineFile, err := a.SavePipelineToFile(yamlContent, newPipeline.PipelineInfo.Name)
	if err != nil {
		result.AddError(fmt.Sprintf("Failed to save pipeline: %v", err))
		return result, err
	}
	result.AddStep(fmt.Sprintf("Saved pipeline to: %s", pipelineFile))

	// Step 8: Upload via MCP server
	result.AddStep("Uploading pipeline via MCP server")
	uploadResult, err := a.UploadPipeline(
		newPipeline.PipelineInfo.Name,
		pipelineFile,
		newPipeline.PipelineInfo.Description,
	)
	if err != nil {
		result.AddError(fmt.Sprintf("Failed to upload pipeline: %v", err))
		return result, err
	}

	if len(uploadResult.Content) > 0 {
		result.AddStep(fmt.Sprintf("Upload successful: %s", uploadResult.Content[0].Text))
	} else {
		result.AddStep("Upload completed successfully")
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = true
	result.PipelineName = newPipeline.PipelineInfo.Name
	result.PipelineFile = pipelineFile
	result.YAMLContent = yamlContent

	log.Printf("Workflow completed successfully in %v", result.Duration)
	return result, nil
}

func getComponentTypeNames(components []ComponentConfig) []string {
	names := make([]string, len(components))
	for i, comp := range components {
		names[i] = comp.Type
	}
	return names
}

type WorkflowStep struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	IsError   bool      `json:"is_error"`
}

type UploadWorkflowResult struct {
	StartTime    time.Time      `json:"start_time"`
	EndTime      time.Time      `json:"end_time"`
	Duration     time.Duration  `json:"duration"`
	Success      bool           `json:"success"`
	Steps        []WorkflowStep `json:"steps"`
	PipelineName string         `json:"pipeline_name,omitempty"`
	PipelineFile string         `json:"pipeline_file,omitempty"`
	YAMLContent  string         `json:"yaml_content,omitempty"`
}

func (r *UploadWorkflowResult) AddStep(message string) {
	r.Steps = append(r.Steps, WorkflowStep{
		Timestamp: time.Now(),
		Message:   message,
		IsError:   false,
	})
	log.Println(message)
}

func (r *UploadWorkflowResult) AddError(message string) {
	r.Steps = append(r.Steps, WorkflowStep{
		Timestamp: time.Now(),
		Message:   message,
		IsError:   true,
	})
	log.Println("ERROR:", message)
}

func (a *AIAgent) Stop() {
	if a.mcpServerStdin != nil {
		a.mcpServerStdin.Close()
	}
	if a.mcpServerStdout != nil {
		a.mcpServerStdout.Close()
	}
	if a.mcpServerCmd != nil {
		a.mcpServerCmd.Process.Kill()
	}
}

// REST API Handlers
func (a *AIAgent) setupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		api.POST("/upload-pipeline", a.handleUploadPipeline)
		api.GET("/component-types", a.handleListComponentTypes)
		api.GET("/health", a.handleHealth)
	}
}

func (a *AIAgent) handleUploadPipeline(c *gin.Context) {
	log.Println("Received upload pipeline request")

	result, err := a.ExecuteUploadWorkflow()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to execute upload workflow",
			"details": err.Error(),
			"result":  result,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Pipeline uploaded successfully",
		"result":  result,
	})
}

func (a *AIAgent) handleListComponentTypes(c *gin.Context) {
	components := a.AnalyzeSDKComponents()
	c.JSON(http.StatusOK, gin.H{
		"components": components,
	})
}

func (a *AIAgent) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
	})
}

func main() {
	agent_port := os.Getenv("AGENT_PORT")
	if agent_port == "" {
		agent_port = "8085"
	}
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	agent := NewAIAgent()
	defer agent.Stop()

	// Start MCP server
	log.Println("Starting MCP server...")
	if err := agent.StartMCPServer(); err != nil {
		log.Fatalf("Failed to start MCP server: %v", err)
	}

	// Wait a moment for MCP server to initialize
	time.Sleep(2 * time.Second)

	// Setup Gin router
	r := gin.Default()

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	agent.setupRoutes(r)

	// Add a simple home page
	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`
<!DOCTYPE html>
<html>
<head>
    <title>KFP AI Agent</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .button { background: #007cba; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; margin: 10px 5px; }
        .button:hover { background: #005a8c; }
        .result { background: #f5f5f5; padding: 15px; border-radius: 4px; margin: 10px 0; white-space: pre-wrap; }
        .error { background: #ffe6e6; border-left: 4px solid #ff0000; }
        .success { background: #e6ffe6; border-left: 4px solid #00aa00; }
    </style>
</head>
<body>
    <h1>Kubeflow Pipelines AI Agent</h1>
    <p>This AI Agent can analyze pipeline files, compare with existing versions, and generate unique pipelines for upload.</p>
    
    <h2>Available Actions:</h2>
    <button class="button" onclick="uploadPipeline()">Upload Pipeline</button>
    <button class="button" onclick="listComponents()">List Component Types</button>
    <button class="button" onclick="checkHealth()">Check Health</button>
    
    <div id="result"></div>
    
    <script>
        function showResult(data, isError = false) {
            const resultDiv = document.getElementById('result');
            resultDiv.className = 'result ' + (isError ? 'error' : 'success');
            resultDiv.textContent = JSON.stringify(data, null, 2);
        }
        
        async function uploadPipeline() {
            try {
                showResult("Starting pipeline upload workflow...");
                const response = await fetch('/api/v1/upload-pipeline', { method: 'POST' });
                const data = await response.json();
                showResult(data, !response.ok);
            } catch (error) {
                showResult({ error: error.message }, true);
            }
        }
        
        async function listComponents() {
            try {
                const response = await fetch('/api/v1/component-types');
                const data = await response.json();
                showResult(data);
            } catch (error) {
                showResult({ error: error.message }, true);
            }
        }
        
        async function checkHealth() {
            try {
                const response = await fetch('/api/v1/health');
                const data = await response.json();
                showResult(data);
            } catch (error) {
                showResult({ error: error.message }, true);
            }
        }
    </script>
</body>
</html>
		`))
	})

	log.Println(fmt.Sprintf("Starting AI Agent server on :%s", agent_port))
	log.Println(fmt.Sprintf("Visit http://localhost:%s to access the web interface", agent_port))

	if err := r.Run(fmt.Sprintf(":%s", agent_port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
