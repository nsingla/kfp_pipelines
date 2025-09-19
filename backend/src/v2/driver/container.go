package driver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
)

// ContainerV2 mirrors Container but uses KFP RunService/ArtifactService instead of MLMD.
// Initial version wires inputs and creates a runtime task; output recording via
// ArtifactService will be added in subsequent steps.
func ContainerV2(ctx context.Context, opts Options, api DriverAPI) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.ContainerV2(%s) failed: %w", opts.info(), err)
		}
	}()
	b, _ := json.Marshal(opts)
	glog.V(4).Info("ContainerV2 opts: ", string(b))
	if api == nil {
		return nil, fmt.Errorf("api client is nil")
	}
	if opts.TaskName == "" {
		return nil, fmt.Errorf("task name flag is required for ContainerV2")
	}
	var iterationIndex *int
	if opts.IterationIndex >= 0 {
		idx := opts.IterationIndex
		iterationIndex = &idx
	}
	expr, err := expression.New()
	if err != nil {
		return nil, err
	}
	inputs, err := resolveInputsV2(ctx, nil, iterationIndex, opts, api, expr)
	if err != nil {
		return nil, err
	}
	executorInput := &pipelinespec.ExecutorInput{Inputs: inputs}
	execution = &Execution{ExecutorInput: executorInput}

	condition := opts.Task.GetTriggerPolicy().GetCondition()
	if condition != "" {
		willTrigger, err := expr.Condition(executorInput, condition)
		if err != nil {
			return execution, err
		}
		execution.Condition = &willTrigger
	}

	pd := &gc.PipelineTaskDetail{
		Name:        opts.TaskName,
		DisplayName: opts.Task.GetTaskInfo().GetName(),
		RunId:       opts.RunID,
		Type:        gc.PipelineTaskDetail_RUNTIME,
	}
	if opts.ParentTaskID != "" {
		pid := opts.ParentTaskID
		pd.ParentTaskId = &pid
	}
	if iterationIndex != nil {
		pd.TypeAttributes = &gc.PipelineTaskDetail_TypeAttributes{IterationIndex: int64(*iterationIndex)}
	}
	_, err = api.CreateTask(ctx, &gc.CreateTaskRequest{Task: pd})
	if err != nil {
		return execution, err
	}

	return execution, nil
}
