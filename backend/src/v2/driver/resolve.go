package driver

import (
	"context"

	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"google.golang.org/protobuf/types/known/structpb"
)

// resolveInputsV2 resolves task inputs using the new KFP APIs (Tasks/Artifacts).
// This is a minimal initial implementation intended to compile and support
// Root DAG and trivial cases. It will be expanded to cover upstream resolution
// (producer tasks, artifact tasks) in subsequent steps.
func resolveInputsV2(
	ctx context.Context,
	parentTask *gc.PipelineTaskDetail,
	iterationIndex *int,
	opts Options,
	api DriverAPI,
	expr *expression.Expr,
) (*pipelinespec.ExecutorInput_Inputs, error) {
	inputs := &pipelinespec.ExecutorInput_Inputs{
		ParameterValues: map[string]*structpb.Value{},
		Artifacts:       map[string]*pipelinespec.ArtifactList{},
	}
	// If parent task is not provided, but we have a parent_task_id, fetch it.
	if parentTask == nil && opts.ParentTaskID != "" {
		pt, err := api.GetTask(ctx, &gc.GetTaskRequest{TaskId: opts.ParentTaskID})
		if err == nil {
			parentTask = pt
		}
	}
	// For the root DAG driver, pass through runtime parameters as inputs to seed the DAG.
	if parentTask != nil && parentTask.GetType() == gc.PipelineTaskDetail_DAG && parentTask.GetParentTaskId() == "" {
		if opts.RuntimeConfig != nil && opts.RuntimeConfig.GetParameterValues() != nil {
			for k, v := range opts.RuntimeConfig.GetParameterValues() {
				inputs.ParameterValues[k] = v
			}
		}
	}
	// TODO: Implement full upstream resolution using:
	// - api.ListTasks with parent filter to discover siblings/ancestors
	// - api.GetTask to fetch details
	// - ArtifactService.ListArtifactTasks to map artifacts produced/consumed
	// - Handle iterationIndex for fan-out
	_ = ctx
	_ = iterationIndex
	_ = api
	_ = expr
	return inputs, nil
}
