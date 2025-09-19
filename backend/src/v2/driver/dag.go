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

// DAGV2 mirrors DAG but uses KFP RunService/ArtifactService instead of MLMD.
// This initial version focuses on wiring and inputs; parent/iteration linkage
// and full upstream resolution will be added incrementally.
func DAGV2(ctx context.Context, opts Options, api DriverAPI) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.DAGV2(%s) failed: %w", opts.info(), err)
		}
	}()
	b, _ := json.Marshal(opts)
	glog.V(4).Info("DAGV2 opts: ", string(b))
	if err = validateDAG(opts); err != nil {
		return nil, err
	}
	if api == nil {
		return nil, fmt.Errorf("api client is nil")
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
	// Parent task resolution TBD; for now, pass nil and support root-level/simple DAGs.
	inputs, err := resolveInputsV2(ctx, nil, iterationIndex, opts, api, expr)
	if err != nil {
		return nil, err
	}
	executorInput := &pipelinespec.ExecutorInput{Inputs: inputs}
	glog.Infof("executorInput value: %+v", executorInput)
	execution = &Execution{ExecutorInput: executorInput}

	condition := opts.Task.GetTriggerPolicy().GetCondition()
	if condition != "" {
		willTrigger, err := expr.Condition(executorInput, condition)
		if err != nil {
			return execution, err
		}
		execution.Condition = &willTrigger
	}

	// Create the DAG task record.
	taskName := opts.TaskName
	if taskName == "" {
		taskName = opts.Task.GetTaskInfo().GetName()
	}

	if opts.TaskName == "" {
		return execution, fmt.Errorf("task name flag is required for DAGV2")
	}

	pd := &gc.PipelineTaskDetail{
		Name:        taskName,
		DisplayName: opts.Task.GetTaskInfo().GetName(),
		RunId:       opts.RunID,
		Type:        gc.PipelineTaskDetail_DAG,
	}
	// Set parent task if provided
	if opts.ParentTaskID != "" {
		pid := opts.ParentTaskID
		pd.ParentTaskId = &pid
	}
	// Set iteration index in TypeAttributes when provided
	if iterationIndex != nil {
		pd.TypeAttributes = &gc.PipelineTaskDetail_TypeAttributes{IterationIndex: int64(*iterationIndex)}
	}

	_, err = api.CreateTask(ctx, &gc.CreateTaskRequest{Task: pd})
	if err != nil {
		return execution, err
	}
	// Iteration fanout count not computed yet in V2 scaffold; preserve behavior fields when possible.
	return execution, nil
}
