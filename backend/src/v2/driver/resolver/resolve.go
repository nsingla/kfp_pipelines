package resolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
)

var paramError = func(paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec, err error) error {
	return fmt.Errorf("resolving input parameter with spec %s: %w", paramSpec, err)
}

var ErrResolvedInputNull = errors.New("the resolved input is null")

type ParameterMetadata struct {
	// This is the key of the parameter in this task's inputs.
	Key                string
	ParameterIO        *apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter
	InputParameterSpec *pipelinespec.TaskInputsSpec_InputParameterSpec
	ParameterIterator  *pipelinespec.ParameterIteratorSpec
}

type ArtifactMetadata struct {
	Key string
	// InputArtifactSpec is mutually exclusive with ArtifactIterator
	InputArtifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec
	ArtifactIterator  *pipelinespec.ArtifactIteratorSpec
	ArtifactIO        *apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact
}

type InputMetadata struct {
	Parameters []ParameterMetadata
	Artifacts  []ArtifactMetadata
}

func ResolveInputs(ctx context.Context, opts common.Options) (*InputMetadata, *int, error) {
	inputMetadata := &InputMetadata{
		Parameters: []ParameterMetadata{},
		Artifacts:  []ArtifactMetadata{},
	}

	var parameterIteratorCount *int
	var artifactIteratorCount *int

	// Handle parameters
	resolvedParameters, parameterIteratorCount, err := resolveParameters(opts)
	if err != nil {
		return nil, nil, err
	}
	inputMetadata.Parameters = resolvedParameters

	// Handle Artifacts
	resolvedArtifacts, artifactIteratorCount, err := resolveArtifacts(ctx, opts)
	if err != nil {
		return nil, nil, err
	}
	inputMetadata.Artifacts = resolvedArtifacts

	// Determine iteration count
	// Note that we can only have one of the two.
	var iterationCount *int
	if parameterIteratorCount != nil && artifactIteratorCount != nil {
		return nil, nil, errors.New("cannot have both parameter and artifact iterators")
	} else if parameterIteratorCount != nil {
		iterationCount = parameterIteratorCount
	} else if artifactIteratorCount != nil {
		iterationCount = artifactIteratorCount
	}

	return inputMetadata, iterationCount, nil
}
