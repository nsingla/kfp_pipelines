package tools

import (
	"context"
	"encoding/json"
	pipeline_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	mcpgolang "github.com/metoro-io/mcp-golang"
	log "github.com/sirupsen/logrus"
)

type ListPipelineVersionsArguments struct {
	pipelineId string `json:"pipeline_id" jsonschema:"required,description=The id of the pipeline for which you want to list all its versions"`
}

func ListPipelines(client *api_server.PipelineClient) (
	[]*pipeline_model.V2beta1Pipeline, int, string, error,
) {
	parameters := &pipeline_params.PipelineServiceListPipelinesParams{}
	logger.Log("Listing all pipelines")
	return client.List(parameters)
}

func ListPipelineVersions(client *api_server.PipelineClient, pipelineId string) []*pipeline_model.V2beta1PipelineVersion {
	logger.Log("Listing pipeline versions for pipeline %s", pipelineId)
	parameters := &pipeline_params.PipelineServiceListPipelineVersionsParams{
		PipelineID: pipelineId,
	}
	versions, _, _, err := client.ListPipelineVersions(parameters)
	if err != nil {
		log.Printf("Error listing pipeline versions for pipeline: %v", err)
	}
	return versions

}

func ListPipelineVersionsHandler(ctx context.Context, client *api_server.PipelineClient, arguments ListPipelineVersionsArguments) (*mcpgolang.ToolResponse, error) {
	var pipelineVersions []*pipeline_model.V2beta1PipelineVersion
	if arguments.pipelineId != "" {
		pipelineVersions = append(pipelineVersions, ListPipelineVersions(client, arguments.pipelineId)...)
	} else {
		pipelines, _, _, pipelineError := ListPipelines(client)
		if pipelineError != nil {
			log.Printf("Error listing pipeline versions for pipeline: %v", pipelineError)
		}
		for _, pipeline := range pipelines {
			pipelineVersions = append(pipelineVersions, ListPipelineVersions(client, pipeline.PipelineID)...)
		}
	}

	outputJson, err := json.Marshal(pipelineVersions)
	if err != nil {
		return nil, err
	}

	return mcpgolang.NewToolResponse(mcpgolang.NewTextContent(string(outputJson))), nil
}
