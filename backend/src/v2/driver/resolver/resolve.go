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
	Key                string
	ParameterIO        *apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter
	InputParameterSpec *pipelinespec.TaskInputsSpec_InputParameterSpec
}

type ArtifactMetadata struct {
	Key               string
	ArtifactIOList    []*apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact
	InputArtifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec
}

type InputMetadata struct {
	Parameters []ParameterMetadata
	Artifacts  []ArtifactMetadata
}

func ResolveInputs(ctx context.Context, opts common.Options) (*InputMetadata, error) {
	inputMetadata := &InputMetadata{
		Parameters: []ParameterMetadata{},
		Artifacts:  []ArtifactMetadata{},
	}
	// Handle parameters
	for name, paramSpec := range opts.Task.GetInputs().GetParameters() {
		if compParam := opts.Component.GetInputDefinitions().GetParameters()[name]; compParam != nil {
			// Skip resolving dsl.TaskConfig because that information is only available after initPodSpecPatch and
			// extendPodSpecPatch are called.
			if compParam.GetParameterType() == pipelinespec.ParameterType_TASK_CONFIG {
				continue
			}
		}

		v, err := resolveInputParameter(opts, paramSpec, opts.ParentTask.Inputs.GetParameters())
		if err != nil {
			if !errors.Is(err, ErrResolvedInputNull) {
				return nil, err
			}
			componentParam, ok := opts.Component.GetInputDefinitions().GetParameters()[name]
			if !ok {
				return nil, fmt.Errorf("parameter %s not found in component input definitions", name)
			}
			// If the resolved parameter was null and the component input parameter is optional, just skip setting
			// it and the launcher will handle defaults.
			if componentParam != nil && componentParam.IsOptional {
				continue
			}
			return nil, err
		}
		inputMetadata.Parameters = append(inputMetadata.Parameters, ParameterMetadata{
			Key:                name,
			ParameterIO:        v,
			InputParameterSpec: paramSpec,
		})
	}

	// Handle artifacts.
	for name, artifactSpec := range opts.Task.GetInputs().GetArtifacts() {
		v, err := resolveInputArtifact(ctx, opts, name, artifactSpec, opts.ParentTask.Inputs.GetArtifacts())
		if err != nil {
			return nil, err
		}
		inputMetadata.Artifacts = append(inputMetadata.Artifacts, ArtifactMetadata{
			Key:               name,
			ArtifactIOList:    v,
			InputArtifactSpec: artifactSpec,
		})
	}

	return inputMetadata, nil
}
