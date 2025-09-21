package driver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"google.golang.org/protobuf/types/known/structpb"
)

// RootDAG handles initial root dag task creation
// and runtime parameter resolution.
func RootDAG(ctx context.Context, opts Options, api DriverAPI) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.RootDAG(%s) failed: %w", opts.info(), err)
		}
	}()
	b, _ := json.Marshal(opts)
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
				Value: val.GetStringValue(),
			})
		}
		inputs = &gc.PipelineTaskDetail_InputOutputs{Parameters: params}
	}
	pd := &gc.PipelineTaskDetail{
		Name:           opts.PipelineName,
		DisplayName:    opts.RunDisplayName,
		RunId:          opts.RunID,
		Type:           gc.PipelineTaskDetail_DAG,
		Inputs:         inputs,
		TypeAttributes: &gc.PipelineTaskDetail_TypeAttributes{},
	}
	_, err = api.CreateTask(ctx, &gc.CreateTaskRequest{Task: pd})
	if err != nil {
		return nil, err
	}
	// Prepare minimal ExecutorInput carrying runtime parameters for downstream resolution.
	executorInput := &structpb.Struct{}
	_ = executorInput // presently unused, kept for parity with old flow
	return &Execution{ID: 0 /* unknown numeric ID in new API world */, ExecutorInput: nil}, nil
}
