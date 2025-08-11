package tools

import (
	"context"
	"encoding/json"
	mcpgolang "github.com/metoro-io/mcp-golang"
)

func ListToolsHandler(ctx context.Context) (*mcpgolang.ToolResponse, error) {

	outputJson, err := json.Marshal(PipelineTools)
	if err != nil {
		return nil, err
	}

	return mcpgolang.NewToolResponse(mcpgolang.NewTextContent(string(outputJson))), nil
}
