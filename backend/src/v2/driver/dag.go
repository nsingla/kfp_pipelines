package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"
	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/resolver"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
)

// DAG mirrors DAG but uses KFP RunService/ArtifactService instead of MLMD.
// This initial version focuses on wiring and inputs; parent/iteration linkage
// and full upstream resolution will be added incrementally.
func DAG(ctx context.Context, opts common.Options, driverAPI common.DriverAPI) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.DAG(%s) failed: %w", opts.Info(), err)
		}
	}()

	b, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	glog.V(4).Info("DAG opts: ", string(b))
	if err = validateDAG(opts); err != nil {
		return nil, err
	}

	if driverAPI == nil {
		return nil, fmt.Errorf("driverAPI client is nil")
	}

	expr, err := expression.New()
	if err != nil {
		return nil, err
	}

	// Determine this Task's Type
	inputs, iterationCount, err := resolver.ResolveInputs(ctx, opts)
	if err != nil {
		return nil, err
	}

	executorInput, err := pipelineTaskInputsToExecutorInputs(inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert inputs to executor inputs: %w", err)
	}

	// TODO(HumairAK) this doesn't seem used in dag case (or root)
	// consider removing it. ExecutorInput is only required by Runtimes.
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

	taskName := opts.TaskName
	if taskName == "" {
		taskName = opts.Task.GetTaskInfo().GetName()
	}
	if opts.TaskName == "" {
		return execution, fmt.Errorf("task name flag is required for DAG")
	}

	if opts.Task.GetArtifactIterator() != nil {
		return execution, fmt.Errorf("ArtifactIterator is not implemented")
	}

	taskToCreate := &gc.PipelineTaskDetail{
		Name:        taskName,
		DisplayName: opts.Task.GetTaskInfo().GetName(),
		RunId:       opts.Run.GetRunId(),
		// Default to DAG
		Type:   gc.PipelineTaskDetail_DAG,
		Status: gc.PipelineTaskDetail_RUNNING,
		Pods: []*gc.PipelineTaskDetail_TaskPod{
			{
				Name: opts.PodName,
				Uid:  opts.PodUID,
				Type: gc.PipelineTaskDetail_DRIVER,
			},
		},
	}

	// Determine type of DAG task.
	// In the future the KFP Sdk should add a Task Type enum to the task Info proto
	// to assist with inferring type. For now, we infer the type based on attribute
	// heuristics.
	if iterationCount != nil {
		iterationCount := int64(*iterationCount)
		taskToCreate.TypeAttributes = &gc.PipelineTaskDetail_TypeAttributes{IterationCount: &iterationCount}
		taskToCreate.Type = gc.PipelineTaskDetail_LOOP
		taskToCreate.DisplayName = "Loop"
	} else if condition != "" {
		taskToCreate.Type = gc.PipelineTaskDetail_CONDITION_BRANCH
		taskToCreate.DisplayName = "Condition Branch"
	} else if strings.HasPrefix(taskName, "condition") && !strings.HasPrefix(taskName, "condition-branch") {
		taskToCreate.Type = gc.PipelineTaskDetail_CONDITION
		taskToCreate.DisplayName = "Condition"
	} else {
		taskToCreate.Type = gc.PipelineTaskDetail_DAG
	}

	if opts.ParentTask.GetTaskId() != "" {
		taskToCreate.ParentTaskId = util.StringPointer(opts.ParentTask.GetTaskId())
	}
	taskToCreate, err = handleTaskParametersCreation(inputs.Parameters, taskToCreate)
	if err != nil {
		return execution, err
	}
	glog.Infof("Creating task: %+v", taskToCreate)
	createdTask, err := driverAPI.CreateTask(ctx, &gc.CreateTaskRequest{Task: taskToCreate})
	if err != nil {
		return execution, err
	}
	glog.Infof("Created task: %+v", createdTask)
	execution.TaskID = createdTask.TaskId

	err = handleTaskArtifactsCreation(ctx, opts, inputs.Artifacts, createdTask, driverAPI)
	if err != nil {
		return execution, err
	}

	return execution, nil
}
