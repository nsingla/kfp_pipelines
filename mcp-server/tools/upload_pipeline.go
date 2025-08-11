package tools

import (
	"context"
	"encoding/json"
	pipeline_upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	mcpgolang "github.com/metoro-io/mcp-golang"
	log "github.com/sirupsen/logrus"
	"strconv"
	"time"
)

type UploadPipelineArguments struct {
	name        string `json:"name" jsonschema:"required,description=The name of the pipeline"`
	displayName string `json:"display_name" jsonschema:"required,description=The display name of the pipeline"`
	namespace   string `json:"namespace" jsonschema:"required,description=The namespace of the pipeline"`
	filePath    string `json:"file_path" jsonschema:"required,description=The file path of the pipeline"`
}

func UploadPipeline(client *api_server.PipelineUploadClient, args UploadPipelineArguments) *pipeline_upload_model.V2beta1Pipeline {
	parameters := &pipeline_upload_params.UploadPipelineParams{
		Name:        &args.name,
		DisplayName: &args.displayName,
		Namespace:   &args.namespace,
	}
	log.Printf("Uploading pipeline")
	pipeline, pipelineErr := client.UploadFile(args.filePath, parameters)
	if pipelineErr != nil {
		log.Printf("Failed to upload pipeline")
	} else {
		return pipeline
	}
	return nil
}

func UploadPipelineHandler(ctx context.Context, client *api_server.PipelineUploadClient) (*mcpgolang.ToolResponse, error) {
	randomName := strconv.FormatInt(time.Now().UnixNano(), 10)
	pipelineGeneratedName := "generated-pipeline-name-" + randomName
	args := UploadPipelineArguments{name: pipelineGeneratedName}
	uploadedPipeline := UploadPipeline(client, args)

	outputJson, err := json.Marshal(uploadedPipeline)
	if err != nil {
		return nil, err
	}

	return mcpgolang.NewToolResponse(mcpgolang.NewTextContent(string(outputJson))), nil
}
