package resolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"google.golang.org/protobuf/types/known/structpb"
)

var paramError = func(paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec, err error) error {
	return fmt.Errorf("resolving input parameter with spec %s: %w", paramSpec, err)
}

var ErrResolvedInputNull = errors.New("the resolved input is null")

func ResolveInputs(
	ctx context.Context,
	iterationIndex *int,
	opts common.Options,
	expr *expression.Expr,
) (*pipelinespec.ExecutorInput_Inputs, error) {
	inputs := &pipelinespec.ExecutorInput_Inputs{
		ParameterValues: make(map[string]*structpb.Value),
		Artifacts:       make(map[string]*pipelinespec.ArtifactList),
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

		v, err := resolveInputParameter(ctx, opts, paramSpec, opts.ParentTask.Inputs.GetParameters())
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
		inputs.ParameterValues[name] = v
	}

	// Handle artifacts.
	for name, artifactSpec := range opts.Task.GetInputs().GetArtifacts() {
		v, err := resolveInputArtifact(ctx, opts, name, artifactSpec, opts.ParentTask.Inputs.GetArtifacts())
		if err != nil {
			return nil, err
		}
		inputs.Artifacts[name] = v
	}

	return inputs, nil
}
