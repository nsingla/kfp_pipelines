package driver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
)

// RootDAG handles initial root dag task creation
// and runtime parameter resolution.
func RootDAG(ctx context.Context, opts Options, api DriverAPI) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.RootDAG(%s) failed: %w", opts.info(), err)
		}
	}()

	b, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	glog.V(4).Info("RootDAG opts: ", string(b))
	if err = validateRootDAG(opts); err != nil {
		return nil, err
	}
	if api == nil {
		return nil, fmt.Errorf("api client is nil")
	}

	// Build minimal PipelineTaskDetail for root DAG task under the run.
	// Inputs: pass runtime parameters into task inputs for record.
	var inputs *gc.PipelineTaskDetail_InputOutputs
	if opts.RuntimeConfig != nil && opts.RuntimeConfig.GetParameterValues() != nil {
		params := make([]*gc.PipelineTaskDetail_InputOutputs_Parameter, 0, len(opts.RuntimeConfig.GetParameterValues()))
		for name, val := range opts.RuntimeConfig.GetParameterValues() {
			n := name
			params = append(params, &gc.PipelineTaskDetail_InputOutputs_Parameter{
				Name:  &n,
				Value: val,
			})
		}
		inputs = &gc.PipelineTaskDetail_InputOutputs{Parameters: params}
	}
	pd := &gc.PipelineTaskDetail{
		Name:           opts.PipelineName,
		DisplayName:    opts.RunDisplayName,
		RunId:          opts.RunID,
		Type:           gc.PipelineTaskDetail_ROOT,
		Inputs:         inputs,
		TypeAttributes: &gc.PipelineTaskDetail_TypeAttributes{},
		Pods: []*gc.PipelineTaskDetail_TaskPod{
			{
				Name: opts.PodName,
				Uid:  opts.PodUID,
				Type: gc.PipelineTaskDetail_DRIVER,
			},
		},
	}
	task, err := api.CreateTask(ctx, &gc.CreateTaskRequest{Task: pd})
	if err != nil {
		return nil, err
	}
	execution = &Execution{
		TaskID: task.TaskId,
	}
	return execution, nil
}
