package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// Container mirrors Container but uses KFP RunService/ArtifactService instead of MLMD.
// Initial version wires inputs and creates a runtime task; output recording via
// ArtifactService will be added in subsequent steps.
func Container(ctx context.Context, opts Options, driverAPI DriverAPI) (execution *Execution, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("driver.Container(%s) failed: %w", opts.info(), err)
		}
	}()
	b, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	glog.V(4).Info("Container opts: ", string(b))

	if driverAPI == nil {
		return nil, fmt.Errorf("driverAPI client is nil")
	}

	if opts.TaskName == "" {
		return nil, fmt.Errorf("task name flag is required for Container")
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

	parentTask, err := driverAPI.GetTask(ctx, &apiV2beta1.GetTaskRequest{TaskId: opts.ParentTaskID})
	if err != nil {
		return nil, err
	}
	opts.ParentTask = parentTask

	// ######################################
	// ### RESOLVE INPUTS ###
	// ######################################
	inputs, err := resolveInputs(ctx, iterationIndex, opts, expr)
	if err != nil {
		return nil, err
	}

	executorInput := &pipelinespec.ExecutorInput{
		Inputs: inputs,
	}
	execution = &Execution{ExecutorInput: executorInput}
	condition := opts.Task.GetTriggerPolicy().GetCondition()
	if condition != "" {
		willTrigger, err := expr.Condition(executorInput, condition)
		if err != nil {
			return execution, err
		}
		execution.Condition = &willTrigger
	}

	// When the container image is a dummy image, there is no launcher for this
	// task. This happens when this task is created to implement a
	// Kubernetes-specific configuration, i.e., there is no user container to
	// run. It publishes execution details to mlmd in driver and takes care of
	// caching, which are usually done in launcher. We also skip creating the
	// podspecpatch in these cases.
	_, isKubernetesPlatformOp := dummyImages[opts.Container.Image]
	if isKubernetesPlatformOp {
		// To be consistent with other artifacts, the driver registers log
		// artifacts to MLMD and the launcher publishes them to the object
		// store. This pattern does not work for kubernetesPlatformOps because
		// they have no launcher. There's no point in registering logs that
		// won't be published. Consequently, when we know we're dealing with
		// kubernetesPlatformOps, we set publishLogs to "false". We can amend
		// this when we update the driver to publish logs directly.
		opts.PublishLogs = "false"
	}

	run, err := driverAPI.GetRun(ctx, &apiV2beta1.GetRunRequest{RunId: opts.Run.GetRunId()})
	if err != nil {
		return nil, err
	}
	if execution.WillTrigger() {
		executorInput.Outputs = provisionOutputs(
			run.RuntimeConfig.PipelineRoot,
			opts.TaskName,
			opts.Component.GetOutputDefinitions(),
			uuid.NewString(),
			opts.PublishLogs,
		)
	}

	// ######################################
	// ### TASK REQUEST ###
	// ######################################
	podName, err := config.InPodName()
	if err != nil {
		return nil, err
	}

	glog.Infof("Creating task %s in pod %s", opts.TaskName, podName)
	taskToCreate := &apiV2beta1.PipelineTaskDetail{
		Name:        opts.TaskName,
		DisplayName: opts.Task.GetTaskInfo().GetName(),
		RunId:       opts.Run.GetRunId(),
		Type:        apiV2beta1.PipelineTaskDetail_RUNTIME,
		Pods: []*apiV2beta1.PipelineTaskDetail_TaskPod{
			{
				Name: opts.PodName,
				Uid:  opts.PodUID,
				Type: apiV2beta1.PipelineTaskDetail_DRIVER,
			},
		},
	}

	if opts.ParentTaskID != "" {
		pid := opts.ParentTaskID
		taskToCreate.ParentTaskId = &pid
	}
	if iterationIndex != nil {
		taskToCreate.TypeAttributes = &apiV2beta1.PipelineTaskDetail_TypeAttributes{IterationIndex: int64(*iterationIndex)}
	}

	// ######################################
	// ### HANDLE K8S OP ###
	// ######################################

	if isKubernetesPlatformOp {
		return execution, kubernetesPlatformOps(ctx, driverAPI, execution, taskToCreate, &opts)
	}

	var inputParams []*apiV2beta1.PipelineTaskDetail_InputOutputs_Parameter
	if opts.KubernetesExecutorConfig != nil {
		inputParams = parentTask.GetInputs().GetParameters()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch input parameters from task: %w", err)
		}
	}

	// ######################################
	// ### CACHE 1 ###
	// ######################################
	var fingerPrint string
	var cachedTask *apiV2beta1.PipelineTaskDetail
	if !opts.CacheDisabled {
		// Generate fingerprint and MLMD ID for cache
		// Start by getting the names of the PVCs that need to be mounted.
		var pvcNames []string
		if opts.KubernetesExecutorConfig != nil && opts.KubernetesExecutorConfig.GetPvcMount() != nil {
			_, volumes, err := makeVolumeMountPatch(
				ctx, opts, opts.KubernetesExecutorConfig.GetPvcMount(),
				inputParams)
			if err != nil {
				return nil, fmt.Errorf("failed to extract volume mount info while generating fingerprint: %w", err)
			}

			for _, volume := range volumes {
				pvcNames = append(pvcNames, volume.Name)
			}
		}

		if needsWorkspaceMount(execution.ExecutorInput) {
			if opts.RunName == "" {
				return execution, fmt.Errorf("failed to generate fingerprint: run name is required when workspace is used")
			}

			pvcNames = append(pvcNames, GetWorkspacePVCName(opts.RunName))
		}

		fingerPrint, cachedTask, err = getFingerPrintsAndID(ctx, execution, driverAPI, &opts, pvcNames)
		if err != nil {
			return execution, err
		}
		taskToCreate.CacheFingerprint = fingerPrint
	}

	// ######################################
	// ### CACHE 1 DONE ###
	// ######################################

	// ######################################
	// ### CREATE TASK ###
	// ######################################

	taskToCreate, err = handleTaskParametersCreation(executorInput, taskToCreate)
	if err != nil {
		return execution, err
	}
	createdTask, err := driverAPI.CreateTask(ctx, &apiV2beta1.CreateTaskRequest{Task: taskToCreate})
	if err != nil {
		return execution, err
	}
	execution.TaskID = createdTask.TaskId

	err = handleTaskArtifactsCreation(ctx, executorInput, opts, createdTask, driverAPI)
	if err != nil {
		return execution, err
	}

	// ######################################
	// ### CACHE 2 ###
	// ######################################

	// Use cache and skip pvc creation if all conditions met:
	// (1) Cache is enabled globally
	// (2) Cache is enabled for the task
	// (3) We had a cache hit for this Task
	execution.Cached = util.BoolPointer(false)
	if !opts.CacheDisabled {
		if opts.Task.GetCachingOptions().GetEnableCache() && cachedTask != nil {
			taskToCreate.Status = apiV2beta1.PipelineTaskDetail_CACHED
			taskToCreate.Outputs = cachedTask.Outputs
			*execution.Cached = true
			_, createErr := driverAPI.UpdateTask(ctx, &apiV2beta1.UpdateTaskRequest{
				Task: taskToCreate,
			})
			if createErr != nil {
				return execution, fmt.Errorf("failed to update task: %w", createErr)
			}
			return execution, nil
		}
	} else {
		glog.Info("Cache disabled globally at the server level.")
	}

	// ######################################
	// ### PodSpecPatch ###
	// ######################################

	taskConfig := &TaskConfig{}

	podSpec, err := initPodSpecPatch(
		opts.Container,
		opts.Component,
		executorInput,
		execution.TaskID,
		opts.PipelineName,
		opts.Run.GetRunId(),
		opts.RunName,
		opts.PipelineLogLevel,
		opts.PublishLogs,
		strconv.FormatBool(opts.CacheDisabled),
		taskConfig,
		fingerPrint,
	)
	if err != nil {
		return execution, err
	}
	if opts.KubernetesExecutorConfig != nil {
		err = extendPodSpecPatch(ctx, podSpec, opts, opts.ParentTask, driverAPI, inputParams, taskConfig)
		if err != nil {
			return execution, err
		}
	}

	// Handle replacing any dsl.TaskConfig inputs with the taskConfig. This is done here because taskConfig is
	// populated by initPodSpecPatch and extendPodSpecPatch.
	taskConfigInputs := map[string]bool{}
	for inputName := range opts.Component.GetInputDefinitions().GetParameters() {
		compParam := opts.Component.GetInputDefinitions().GetParameters()[inputName]
		if compParam != nil && compParam.GetParameterType() == pipelinespec.ParameterType_TASK_CONFIG {
			taskConfigInputs[inputName] = true
		}
	}

	if len(taskConfigInputs) > 0 {
		taskConfigBytes, err := json.Marshal(taskConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Kubernetes passthrough info: %w", err)
		}

		taskConfigStruct := &structpb.Struct{}
		err = protojson.Unmarshal(taskConfigBytes, taskConfigStruct)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal Kubernetes passthrough info: %w", err)
		}

		for inputName := range taskConfigInputs {
			executorInput.Inputs.ParameterValues[inputName] = &structpb.Value{
				Kind: &structpb.Value_StructValue{StructValue: taskConfigStruct},
			}
		}

		// Overwrite the --executor_input argument in the podSpec container command with the updated executorInput
		executorInputJSON, err := protojson.Marshal(executorInput)
		if err != nil {
			return execution, fmt.Errorf("JSON marshaling executor input: %w", err)
		}
		for index, container := range podSpec.Containers {
			if container.Name == "main" {
				cmd := container.Command
				for i := 0; i < len(cmd)-1; i++ {
					if cmd[i] == "--executor_input" {
						podSpec.Containers[index].Command[i+1] = string(executorInputJSON)
						break
					}
				}
				break
			}
		}
		execution.ExecutorInput = executorInput
	}

	podSpecPatchBytes, err := json.Marshal(podSpec)
	if err != nil {
		return execution, fmt.Errorf("JSON marshaling pod spec patch: %w", err)
	}
	execution.PodSpecPatch = string(podSpecPatchBytes)
	return execution, nil
}
