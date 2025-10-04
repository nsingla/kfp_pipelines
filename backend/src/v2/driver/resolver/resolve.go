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

	// Handle parameters
	resolvedParameters, err := resolveParameters(opts)
	if err != nil {
		return nil, nil, err
	}
	inputMetadata.Parameters = resolvedParameters

	// Handle Artifacts
	resolvedArtifacts, err := resolveArtifacts(ctx, opts)
	if err != nil {
		return nil, nil, err
	}
	inputMetadata.Artifacts = resolvedArtifacts

	// Note that we can only have one of the two.
	var iterationCount *int
	artifactIterator := opts.Task.GetArtifactIterator()
	parameterIterator := opts.Task.GetParameterIterator()
	if parameterIterator != nil && artifactIterator != nil {
		return nil, nil, errors.New("cannot have both parameter and artifact iterators")
	} else if parameterIterator != nil {
		pm, count, err := resolveParameterIterator(opts, inputMetadata.Parameters)
		if err != nil {
			return nil, nil, err
		}
		if len(pm) == 0 {
			return nil, nil, fmt.Errorf("parameter iterator is empty")
		}
		iterationCount = count
		inputMetadata.Parameters = append(inputMetadata.Parameters, pm...)
	} else if artifactIterator != nil {
		am, count, err := resolveArtifactIterator(opts, inputMetadata.Artifacts)
		if err != nil {
			return nil, nil, err
		}
		if len(am) == 0 {
			return nil, nil, fmt.Errorf("artifact iterator is empty")
		}
		iterationCount = count
		inputMetadata.Artifacts = append(inputMetadata.Artifacts, am...)
	}

	return inputMetadata, iterationCount, nil
}
