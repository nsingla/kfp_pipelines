package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"google.golang.org/protobuf/types/known/structpb"
)

// DAG mirrors DAG but uses KFP RunService/ArtifactService instead of MLMD.
// This initial version focuses on wiring and inputs; parent/iteration linkage
// and full upstream resolution will be added incrementally.
func DAG(ctx context.Context, opts Options, driverAPI DriverAPI) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.DAG(%s) failed: %w", opts.info(), err)
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

	var iterationIndex *int
	if opts.IterationIndex >= 0 {
		idx := opts.IterationIndex
		iterationIndex = &idx
	}

	expr, err := expression.New()
	if err != nil {
		return nil, err
	}

	// TODO(HumairAK): Do we need this?
	//var parentTask *gc.PipelineTaskDetail
	//if opts.ParentTaskID != "" {
	//	glog.Infof("Parent task ID: %s", opts.ParentTaskID)
	//	getTaskRequest := &gc.GetTaskRequest{
	//		TaskId: opts.ParentTaskID,
	//	}
	//	parentTask, err = driverAPI.GetTask(ctx, getTaskRequest)
	//	if err != nil {
	//		return nil, err
	//	}
	//}

	inputs, err := resolveInputs(ctx, iterationIndex, opts, expr)
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
		return execution, fmt.Errorf("task name flag is required for DAG")
	}

	if opts.Task.GetArtifactIterator() != nil {
		return execution, fmt.Errorf("ArtifactIterator is not implemented")
	}
	isIterator := opts.Task.GetParameterIterator() != nil && opts.IterationIndex < 0
	// Fan out iterations
	var iterationCount *int
	if execution.WillTrigger() && isIterator {
		iterator := opts.Task.GetParameterIterator()
		report := func(err error) error {
			return fmt.Errorf("iterating on item input %q failed: %w", iterator.GetItemInput(), err)
		}
		// Check the items type of parameterIterator:
		// It can be "inputParameter" or "Raw"
		var value *structpb.Value
		switch iterator.GetItems().GetKind().(type) {
		case *pipelinespec.ParameterIteratorSpec_ItemsSpec_InputParameter:
			var ok bool
			value, ok = executorInput.GetInputs().GetParameterValues()[iterator.GetItems().GetInputParameter()]
			if !ok {
				return execution, report(fmt.Errorf("cannot find input parameter"))
			}
		case *pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw:
			valueRaw := iterator.GetItems().GetRaw()
			var unmarshalledRaw interface{}
			err = json.Unmarshal([]byte(valueRaw), &unmarshalledRaw)
			if err != nil {
				return execution, fmt.Errorf("error unmarshall raw string: %q", err)
			}
			value, err = structpb.NewValue(unmarshalledRaw)
			if err != nil {
				return execution, fmt.Errorf("error converting unmarshalled raw string into protobuf Value type: %q", err)
			}
			// Add the raw input to the executor input
			execution.ExecutorInput.Inputs.ParameterValues[iterator.GetItemInput()] = value
		default:
			return execution, fmt.Errorf("cannot find parameter iterator")
		}
		items, err := getItems(value)
		if err != nil {
			return execution, report(err)
		}
		count := len(items)
		iterationCount = &count
		execution.IterationCount = &count
	}

	taskToCreate := &gc.PipelineTaskDetail{
		Name:        taskName,
		DisplayName: opts.Task.GetTaskInfo().GetName(),
		RunId:       opts.RunID,
		// Default to DAG
		Type: gc.PipelineTaskDetail_DAG,
		Pods: []*gc.PipelineTaskDetail_TaskPod{
			{
				Name: opts.PodName,
				Uid:  opts.PodUID,
				Type: gc.PipelineTaskDetail_DRIVER,
			},
		},
	}
	// TODO(HumairAK): Add conversion from executor input to dag task inputs

	// TODO(HumairAK): Create artifact_tasks for resolved artifacts

	// Determine type of DAG task.
	// In the future the KFP Sdk should add a Task Type enum to the task Info proto
	// to assist with inferring type. For now we infer the type based on attribute
	// heuristics.
	if iterationCount != nil {
		taskToCreate.TypeAttributes = &gc.PipelineTaskDetail_TypeAttributes{IterationCount: int64(*iterationCount)}
		taskToCreate.Type = gc.PipelineTaskDetail_LOOP
		taskToCreate.DisplayName = "Loop"
	} else if iterationIndex != nil {
		taskToCreate.TypeAttributes = &gc.PipelineTaskDetail_TypeAttributes{IterationIndex: int64(*iterationIndex)}
		taskToCreate.Type = gc.PipelineTaskDetail_LOOP_ITERATION
		taskToCreate.DisplayName = fmt.Sprintf("Loop Iteration %d", *iterationIndex)
	} else if condition != "" {
		taskToCreate.Type = gc.PipelineTaskDetail_CONDITION_BRANCH
		taskToCreate.DisplayName = "Condition Branch"
	} else if strings.HasPrefix(taskName, "condition") && !strings.HasPrefix(taskName, "condition-branch") {
		taskToCreate.Type = gc.PipelineTaskDetail_CONDITION
		taskToCreate.DisplayName = "Condition"
	}

	if opts.ParentTaskID != "" {
		taskToCreate.ParentTaskId = &opts.ParentTaskID
	}
	taskToCreate, err = handleTaskParametersCreation(executorInput, taskToCreate)
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

	err = handleTaskArtifactsCreation(ctx, executorInput, opts, createdTask, driverAPI)
	if err != nil {
		return execution, err
	}

	return execution, nil
}
