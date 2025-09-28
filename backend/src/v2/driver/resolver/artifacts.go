package resolver

import (
	"context"
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
)

func resolveInputArtifact(
	ctx context.Context,
	opts common.Options,
	name string,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
) (*pipelinespec.ArtifactList, error) {
	artifactError := func(err error) error {
		return fmt.Errorf("failed to resolve input artifact %s with spec %s: %w", name, artifactSpec, err)
	}
	switch t := artifactSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_ComponentInputArtifact:
		// TODO(HumairAK)
		return nil, fmt.Errorf("not implemented yet")
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact:
		artifacts, err := resolveUpstreamArtifacts(ctx, opts, name, artifactSpec)
		if err != nil {
			return nil, err
		}
		return artifacts, nil
	default:
		return nil, artifactError(fmt.Errorf("artifact spec of type %T not implemented yet", t))
	}
}

func resolveUpstreamArtifacts(ctx context.Context,
	opts common.Options,
	name string,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec) (*pipelinespec.ArtifactList, error) {

	return &pipelinespec.ArtifactList{}, nil
}
