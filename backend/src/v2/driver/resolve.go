// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	removethis "github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

var ErrResolvedParameterNull = errors.New("the resolved input parameter is null")

// resolveUpstreamOutputsConfig is just a config struct used to store the input
// parameters of the resolveUpstreamParameters and resolveUpstreamArtifacts
// functions.
type resolveUpstreamOutputsConfig struct {
	ctx          context.Context
	paramSpec    *pipelinespec.TaskInputsSpec_InputParameterSpec
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec
	opts         Options
	err          func(error) error
}

func generateUniqueTaskName(task, parentTask *apiv2beta1.PipelineTaskDetail) (string, error) {
	if task == nil || task.Name == "" {
		return "", fmt.Errorf("task can't be nil or name cannot be empty")
	}
	taskName := fmt.Sprintf("%s_%s", task.Name, task.TaskId)

	if task.Type == apiv2beta1.PipelineTaskDetail_LOOP_ITERATION {
		if task.TypeAttributes == nil || task.TypeAttributes.IterationIndex == nil {
			return "", fmt.Errorf("iteration index cannot be nil for loop iteration")
		}
		taskName = getParallelForTaskName(taskName, *task.TypeAttributes.IterationIndex)
	} else if parentTask != nil && parentTask.Type == apiv2beta1.PipelineTaskDetail_LOOP_ITERATION {
		if parentTask.TypeAttributes == nil || parentTask.TypeAttributes.IterationIndex == nil {
			return "", fmt.Errorf("iteration index cannot be nil for loop iteration")
		}
		taskName = fmt.Sprintf("%s_idx_%d", taskName, parentTask.TypeAttributes.IterationIndex)
	}
	return taskName, nil
}

func getChildTasks(
	tasks []*apiv2beta1.PipelineTaskDetail,
	parentTask *apiv2beta1.PipelineTaskDetail,
) (map[string]*apiv2beta1.PipelineTaskDetail, error) {
	if parentTask == nil {
		return nil, fmt.Errorf("parent task cannot be nil")
	}
	var taskMap map[string]*apiv2beta1.PipelineTaskDetail
	for _, task := range tasks {
		if task.ParentTaskId == parentTask.ParentTaskId {
			taskName, err := generateUniqueTaskName(task, parentTask)
			if err != nil {
				return nil, err
			}
			if taskName == "" {
				return nil, fmt.Errorf("task name cannot be empty")
			}
			taskMap[taskName] = task
		}
	}
	return taskMap, nil
}

func getSubTasks(
	currentTask *apiv2beta1.PipelineTaskDetail,
	allRuntasks []*apiv2beta1.PipelineTaskDetail,
	flattenedTasks map[string]*apiv2beta1.PipelineTaskDetail,
) (map[string]*apiv2beta1.PipelineTaskDetail, error) {

	if flattenedTasks == nil {
		flattenedTasks = make(map[string]*apiv2beta1.PipelineTaskDetail)
	}

	taskChildren, err := getChildTasks(allRuntasks, currentTask)
	if err != nil {
		return nil, fmt.Errorf("failed to get child tasks for task %s: %w", currentTask.Name, err)
	}
	for taskName, task := range taskChildren {
		flattenedTasks[taskName] = task
	}
	for _, task := range taskChildren {
		if task.Type != apiv2beta1.PipelineTaskDetail_RUNTIME {
			flattenedTasks, err = getSubTasks(task, allRuntasks, flattenedTasks)
			if err != nil {
				return nil, err
			}
		}
	}
	return flattenedTasks, nil
}

func convertTaskInputParamsToExecutorInputParams(
	params []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter,
) (map[string]*structpb.Value, error) {
	convertedParams := make(map[string]*structpb.Value)
	for _, p := range params {
		if p.GetParameterName() != "" && p.GetProducer() != nil {
			return nil, fmt.Errorf("cannot have both parameter name and producer")
		}
		name, err := parseIONameOrPipelineChannel(p.GetParameterName(), p.GetProducer())
		if err != nil {
			return nil, err
		}
		convertedParams[name] = p.GetValue()
	}
	return convertedParams, nil
}

func convertTaskInputArtifactsToExecutorInputArtifacts(
	artifacts []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact,
) (map[string]*pipelinespec.ArtifactList, error) {
	var convertedArtifactsMap = make(map[string]*pipelinespec.ArtifactList)
	for _, artifactIO := range artifacts {
		var convertedRuntimeArtifacts []*pipelinespec.RuntimeArtifact
		for _, artifact := range artifactIO.Artifacts {
			if artifact.GetName() == "" && artifact.GetUri() == "" {
				return nil, fmt.Errorf("artifact name or uri cannot be empty")
			}
			runtimeArtifact := &pipelinespec.RuntimeArtifact{
				Name: artifact.GetName(),
				Type: &pipelinespec.ArtifactTypeSchema{
					Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: artifact.Type.String()},
				},
			}
			if artifact.GetUri() != "" {
				runtimeArtifact.Uri = artifact.GetUri()
			}
			if artifact.GetMetadata() != nil {
				runtimeArtifact.Metadata = &structpb.Struct{
					Fields: artifact.GetMetadata(),
				}
			}
			convertedRuntimeArtifacts = append(convertedRuntimeArtifacts, runtimeArtifact)
		}
		name, err := parseIONameOrPipelineChannel(artifactIO.GetParameterName(), artifactIO.GetProducer())
		if err != nil {
			return nil, err
		}
		convertedArtifactsMap[name] = &pipelinespec.ArtifactList{
			Artifacts: convertedRuntimeArtifacts,
		}
	}
	return convertedArtifactsMap, nil
}

func resolveInputs(
	ctx context.Context,
	iterationIndex *int,
	opts Options,
	expr *expression.Expr,
) (inputs *pipelinespec.ExecutorInput_Inputs, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to resolve inputs: %w", err)
		}
	}()

	task := opts.Task
	inputsSpec := opts.Component.GetInputDefinitions()

	glog.V(4).Infof("task: %v", task)
	glog.Infof("parent DAG input parameters: %+v, artifacts: %+v",
		opts.ParentTask.Inputs.GetParameters(), opts.ParentTask.Inputs.GetArtifacts())

	inputParams, err := convertTaskInputParamsToExecutorInputParams(opts.ParentTask.Inputs.GetParameters())
	if err != nil {
		return nil, err
	}
	inputArtifacts, err := convertTaskInputArtifactsToExecutorInputArtifacts(opts.ParentTask.Inputs.GetArtifacts())
	if err != nil {
		return nil, err
	}
	inputs = &pipelinespec.ExecutorInput_Inputs{
		ParameterValues: make(map[string]*structpb.Value),
		Artifacts:       make(map[string]*pipelinespec.ArtifactList),
	}
	isIterationDriver := iterationIndex != nil

	handleParameterExpressionSelector := func() error {
		for name, paramSpec := range task.GetInputs().GetParameters() {
			var selector string
			if selector = paramSpec.GetParameterExpressionSelector(); selector == "" {
				continue
			}
			wrap := func(e error) error {
				return fmt.Errorf("resolving parameter %q: evaluation of parameter expression selector %q failed: %w", name, selector, e)
			}
			value, ok := inputs.ParameterValues[name]
			if !ok {
				return wrap(fmt.Errorf("value not found in inputs"))
			}
			selected, err := expr.Select(value, selector)
			if err != nil {
				return wrap(err)
			}
			inputs.ParameterValues[name] = selected
		}
		return nil
	}
	// Track parameters set to nil by the driver (for the case in which optional pipeline input parameters are not
	// included, and default value is nil).
	parametersSetNilByDriver := map[string]bool{}
	handleParamTypeValidationAndConversion := func() error {
		// TODO(Bobgy): verify whether there are inputs not in the inputs spec.
		for name, spec := range inputsSpec.GetParameters() {
			if task.GetParameterIterator() != nil {
				if !isIterationDriver && task.GetParameterIterator().GetItemInput() == name {
					// It's expected that an iterator does not have iteration item input parameter,
					// because only iterations get the item input parameter.
					continue
				}
				if isIterationDriver && task.GetParameterIterator().GetItems().GetInputParameter() == name {
					// It's expected that an iteration does not have iteration items input parameter,
					// because only the iterator has it.
					continue
				}
			}
			value, hasValue := inputs.GetParameterValues()[name]

			// Handle when parameter does not have input value
			if !hasValue && !inputsSpec.GetParameters()[name].GetIsOptional() {
				// When parameter is not optional and there is no input value, first check if there is a default value,
				// if there is a default value, use it as the value of the parameter.
				// if there is no default value, report error.
				if inputsSpec.GetParameters()[name].GetDefaultValue() == nil {
					return fmt.Errorf("neither value nor default value provided for non-optional parameter %q", name)
				}
			} else if !hasValue && inputsSpec.GetParameters()[name].GetIsOptional() {
				// When parameter is optional and there is no input value, value comes from default value.
				// But we don't pass the default value here. They are resolved internally within the component.
				// Note: in the past the backend passed the default values into the component. This is a behavior change.
				// See discussion: https://github.com/kubeflow/pipelines/pull/8765#discussion_r1119477085
				continue
			}

			switch spec.GetParameterType() {
			case pipelinespec.ParameterType_STRING:
				_, isValueString := value.GetKind().(*structpb.Value_StringValue)
				if !isValueString {
					// If parameter was set to nil by driver, allow input parameter to have a nil value.
					if parametersSetNilByDriver[name] {
						continue
					}
					// TODO(Bobgy): discuss whether we want to allow auto type conversion
					// all parameter types can be consumed as JSON string
					text, err := metadata.PbValueToText(value)
					if err != nil {
						return fmt.Errorf("converting input parameter %q to string: %w", name, err)
					}
					inputs.GetParameterValues()[name] = structpb.NewStringValue(text)
				}
			default:
				typeMismatch := func(actual string) error {
					return fmt.Errorf("input parameter %q type mismatch: expect %s, got %s", name, spec.GetParameterType(), actual)
				}
				switch v := value.GetKind().(type) {
				case *structpb.Value_NullValue:
					// If parameter was set to nil by driver, allow input parameter to have a nil value.
					if parametersSetNilByDriver[name] {
						continue
					}
					return fmt.Errorf("got null for input parameter %q", name)
				case *structpb.Value_StringValue:
					// TODO(Bobgy): consider whether we support parsing string as JSON for any other types.
					if spec.GetParameterType() != pipelinespec.ParameterType_STRING {
						return typeMismatch("string")
					}
				case *structpb.Value_NumberValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_NUMBER_DOUBLE && spec.GetParameterType() != pipelinespec.ParameterType_NUMBER_INTEGER {
						return typeMismatch("number")
					}
				case *structpb.Value_BoolValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_BOOLEAN {
						return typeMismatch("bool")
					}
				case *structpb.Value_ListValue:
					if spec.GetParameterType() != pipelinespec.ParameterType_LIST {
						return typeMismatch("list")
					}
				case *structpb.Value_StructValue:
					if (spec.GetParameterType() != pipelinespec.ParameterType_STRUCT) && (spec.GetParameterType() != pipelinespec.ParameterType_TASK_FINAL_STATUS) && (spec.GetParameterType() != pipelinespec.ParameterType_TASK_CONFIG) {
						return typeMismatch("struct")
					}
				default:
					return fmt.Errorf("parameter %s has unknown protobuf.Value type: %T", name, v)
				}
			}
		}
		return nil
	}
	defer func() {
		if err == nil {
			err = handleParameterExpressionSelector()
		}
		if err == nil {
			err = handleParamTypeValidationAndConversion()
		}
	}()

	// resolve input parameters
	if isIterationDriver {
		// resolve inputs for iteration driver is very different
		inputs.ParameterValues = inputParams
		inputs.Artifacts = inputArtifacts
		switch {
		case task.GetArtifactIterator() != nil:
			return nil, fmt.Errorf("artifact iterator not implemented yet")
		case task.GetParameterIterator() != nil:
			var itemsInput string
			if task.GetParameterIterator().GetItems().GetInputParameter() != "" {
				// input comes from outside the component
				itemsInput = task.GetParameterIterator().GetItems().GetInputParameter()
			} else if task.GetParameterIterator().GetItemInput() != "" {
				// input comes from static input
				itemsInput = task.GetParameterIterator().GetItemInput()
			} else {
				return nil, fmt.Errorf("cannot retrieve parameter iterator")
			}
			// Inside an iteration (ParallelFor) driver, the task should receive only
			// the single element for the current iteration, not the entire collection
			// it’s iterating over. We first retrieve all the elements from the collection.
			items, err := getItems(inputs.ParameterValues[itemsInput])
			if err != nil {
				return nil, err
			}
			if iterationIndex == nil {
				return nil, fmt.Errorf("iteration_index is nil for iterator")
			}
			if *iterationIndex >= len(items) {
				return nil, fmt.Errorf("bug: %v items found, but getting index %v", len(items), *iterationIndex)
			}
			// Then we replace the list input parameter with the single element for the current iteration.
			delete(inputs.ParameterValues, itemsInput)
			inputs.ParameterValues[task.GetParameterIterator().GetItemInput()] = items[*iterationIndex]
		default:
			return nil, fmt.Errorf("bug: iteration_index>=0, but task iterator is empty")
		}
		return inputs, nil
	}

	// A DAG driver (not Root DAG driver) indicates this is likely the start of a nested pipeline.
	// Handle omitted optional pipeline input parameters similar to how they are handled on the root pipeline.
	isDagDriver := opts.DriverType == "DAG"
	if isDagDriver {
		for name, paramSpec := range opts.Component.GetInputDefinitions().GetParameters() {
			_, ok := task.Inputs.GetParameters()[name]
			if !ok && paramSpec.IsOptional {
				if paramSpec.GetDefaultValue() != nil {
					// If no value was input, pass along the default value to the component.
					inputs.ParameterValues[name] = paramSpec.GetDefaultValue()
				} else {
					//  If no default value is set, pass along the null value to the component.
					//	This is analogous to a pipeline run being submitted without optional pipeline input parameters.
					inputs.ParameterValues[name] = structpb.NewNullValue()
					parametersSetNilByDriver[name] = true
				}
			}
		}
	}
	// Handle parameters.
	for name, paramSpec := range task.GetInputs().GetParameters() {
		if compParam := opts.Component.GetInputDefinitions().GetParameters()[name]; compParam != nil {
			// Skip resolving dsl.TaskConfig because that information is only available after initPodSpecPatch and
			// extendPodSpecPatch are called.
			if compParam.GetParameterType() == pipelinespec.ParameterType_TASK_CONFIG {
				continue
			}
		}
		v, err := resolveInputParameter(ctx, opts, paramSpec, inputParams)
		if err != nil {
			if !errors.Is(err, ErrResolvedParameterNull) {
				return nil, err
			}

			componentParam, ok := opts.Component.GetInputDefinitions().GetParameters()[name]
			if ok && componentParam != nil && componentParam.IsOptional {
				// If the resolved paramter was null and the component input parameter is optional, just skip setting
				// it and the launcher will handle defaults.
				continue
			}

			return nil, err
		}

		inputs.ParameterValues[name] = v
	}

	// Handle artifacts.
	for name, artifactSpec := range task.GetInputs().GetArtifacts() {
		v, err := resolveInputArtifact(ctx, opts, name, artifactSpec, inputArtifacts, task)
		if err != nil {
			return nil, err
		}
		inputs.Artifacts[name] = v
	}

	return inputs, nil
}

func fetchInputParam(
	paramName string,
	inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter) (*structpb.Value, error) {
	for _, param := range inputParams {
		generateName, err := parseIONameOrPipelineChannel(param.GetParameterName(), param.GetProducer())
		if err != nil {
			return nil, err
		}
		if paramName == generateName {
			return param.GetValue(), nil
		}
	}
	return nil, fmt.Errorf("failed to find input param %s", paramName)
}

func fetchTask(taskID string, tasks []*apiv2beta1.PipelineTaskDetail) (*apiv2beta1.PipelineTaskDetail, error) {
	for _, t := range tasks {
		if t.GetTaskId() == taskID {
			return t, nil
		}
	}
	return nil, fmt.Errorf("failed to find task %s", taskID)
}

// GetTaskNameWithTaskID appends the taskName with its parent dag id. This is
// used to help avoid collisions when creating the taskMap for downstream input
// resolution.
func getTaskNameWithTaskID(taskName, taskID string) string {
	return fmt.Sprintf("%s_%s", taskName, taskID)
}

// resolveInputParameter resolves an InputParameterSpec
// using a given input context via InputParams. ErrResolvedParameterNull is returned if paramSpec
// is a component input parameter and parameter resolves to a null value (i.e. an optional pipeline input with no
// default). The caller can decide if this is allowed in that context.
func resolveInputParameter(
	ctx context.Context,
	opts Options,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter,
) (*structpb.Value, error) {
	glog.V(4).Infof("paramSpec: %v", paramSpec)
	paramError := func(err error) error {
		return fmt.Errorf("resolving input parameter with spec %s: %w", paramSpec, err)
	}
	switch t := paramSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter:
		componentInput := paramSpec.GetComponentInputParameter()
		if componentInput == "" {
			return nil, paramError(fmt.Errorf("empty component input"))
		}
		v, err := fetchInputParam(componentInput, inputParams)
		if err != nil {
			return nil, paramError(fmt.Errorf("parent Task does not have input parameter %s", componentInput))
		}

		if _, isNullValue := v.GetKind().(*structpb.Value_NullValue); isNullValue {
			// Null values are only allowed for optional pipeline input parameters with no values. The caller has this
			// context to know if this is allowed.
			return nil, fmt.Errorf("%w: %s", ErrResolvedParameterNull, componentInput)
		}

		return v, nil

	// This is the case where the input comes from the output of an upstream task.
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:
		cfg := resolveUpstreamOutputsConfig{
			ctx:       ctx,
			paramSpec: paramSpec,
			err:       paramError,
			opts:      opts,
		}
		v, err := resolveUpstreamParameters(cfg)
		if err != nil {
			return nil, err
		}
		return v, nil
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue:
		runtimeValue := paramSpec.GetRuntimeValue()
		switch t := runtimeValue.Value.(type) {
		case *pipelinespec.ValueOrRuntimeParameter_Constant:
			val := runtimeValue.GetConstant()
			valStr := val.GetStringValue()
			var v *structpb.Value

			if strings.Contains(valStr, "{{$.workspace_path}}") {
				v = structpb.NewStringValue(strings.ReplaceAll(valStr, "{{$.workspace_path}}", component.WorkspaceMountPath))
				return v, nil
			}

			switch valStr {
			case "{{$.pipeline_job_name}}":
				v = structpb.NewStringValue(opts.RunDisplayName)
			case "{{$.pipeline_job_resource_name}}":
				v = structpb.NewStringValue(opts.RunName)
			case "{{$.pipeline_job_uuid}}":
				v = structpb.NewStringValue(opts.Run.GetRunId())
			case "{{$.pipeline_task_name}}":
				v = structpb.NewStringValue(opts.TaskName)
			// TODO(HumairAK): Shouldn't this be the name of the Runtime Task UUID ?
			case "{{$.pipeline_task_uuid}}":
				v = structpb.NewStringValue(fmt.Sprintf("%s", opts.ParentTaskID))
			default:
				v = val
			}
			return v, nil
		default:
			return nil, paramError(fmt.Errorf("param runtime value spec of type %T not implemented", t))
		}
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_:
		tasks, err := getSubTasks(opts.ParentTask, opts.Run.Tasks, nil)
		if err != nil {
			return nil, err
		}

		if len(opts.Task.DependentTasks) < 1 {
			return nil, fmt.Errorf("task %v has no dependent tasks", opts.Task.TaskInfo.GetName())
		}
		producer, ok := tasks[getTaskNameWithTaskID(opts.Task.DependentTasks[0], opts.ParentTask.GetTaskId())]
		if !ok {
			return nil, fmt.Errorf("producer task, %s, not in tasks", producer.GetName())
		}
		finalStatus := pipelinespec.PipelineTaskFinalStatus{
			State:                   producer.GetStatus().String(),
			PipelineTaskName:        producer.GetName(),
			PipelineJobResourceName: opts.RunName,
			// TODO: Implement fields "Message and "Code" below for Error status.
			Error: &status.Status{},
		}
		finalStatusJSON, err := protojson.Marshal(&finalStatus)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal PipelineTaskFinalStatus: %w", err)
		}

		var finalStatusMap map[string]interface{}
		if err := json.Unmarshal(finalStatusJSON, &finalStatusMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON of PipelineTaskFinalStatus: %w", err)
		}

		finalStatusStruct, err := structpb.NewStruct(finalStatusMap)
		if err != nil {
			return nil, fmt.Errorf("failed to create structpb.Struct: %w", err)
		}

		return structpb.NewStructValue(finalStatusStruct), nil
	default:
		return nil, paramError(fmt.Errorf("parameter spec of type %T not implemented yet", t))
	}
}

// resolveInputParameterStr is like resolveInputParameter but returns an error if the resolved value is not a non-empty
// string.
func resolveInputParameterStr(
	ctx context.Context,
	opts Options,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter,
) (*structpb.Value, error) {
	val, err := resolveInputParameter(ctx, opts, paramSpec, inputParams)
	if err != nil {
		return nil, err
	}

	if typedVal, ok := val.GetKind().(*structpb.Value_StringValue); ok && typedVal != nil {
		if typedVal.StringValue == "" {
			return nil, fmt.Errorf("resolving input parameter with spec %s. Expected a non-empty string", paramSpec)
		}
	} else {
		return nil, fmt.Errorf("resolving input parameter with spec %s. Expected a string but got: %T", paramSpec, val.GetKind())
	}

	return val, nil
}

// resolveInputArtifact resolves an InputArtifactSpec
// using a given input context via inputArtifacts.
func resolveInputArtifact(
	ctx context.Context,
	opts Options,
	name string,
	artifactSpec *pipelinespec.TaskInputsSpec_InputArtifactSpec,
	inputArtifacts map[string]*pipelinespec.ArtifactList,
	task *pipelinespec.PipelineTaskSpec,
) (*pipelinespec.ArtifactList, error) {
	glog.V(4).Infof("inputs: %#v", task.GetInputs())
	glog.V(4).Infof("artifacts: %#v", task.GetInputs().GetArtifacts())
	artifactError := func(err error) error {
		return fmt.Errorf("failed to resolve input artifact %s with spec %s: %w", name, artifactSpec, err)
	}
	switch t := artifactSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_ComponentInputArtifact:
		inputArtifactName := artifactSpec.GetComponentInputArtifact()
		if inputArtifactName == "" {
			return nil, artifactError(fmt.Errorf("component input artifact key is empty"))
		}
		v, ok := inputArtifacts[inputArtifactName]
		if !ok {
			return nil, artifactError(fmt.Errorf("parent DAG does not have input artifact %s", inputArtifactName))
		}
		return v, nil
	case *pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact:
		cfg := resolveUpstreamOutputsConfig{
			ctx:          ctx,
			artifactSpec: artifactSpec,
			opts:         opts,
			err:          artifactError,
		}
		artifacts, err := resolveUpstreamArtifacts(cfg)
		if err != nil {
			return nil, err
		}
		return artifacts, nil
	default:
		return nil, artifactError(fmt.Errorf("artifact spec of type %T not implemented yet", t))
	}
}

// resolveUpstreamParameters resolves input parameters that come from upstream
// tasks. These tasks can be components/containers, which is relatively
// straightforward, or DAGs, in which case, we need to traverse the graph until
// we arrive at a component/container (since there can be n nested DAGs).
func resolveUpstreamParameters(cfg resolveUpstreamOutputsConfig) (*structpb.Value, error) {
	taskOutput := cfg.paramSpec.GetTaskOutputParameter()
	glog.V(4).Info("taskOutput: ", taskOutput)
	producerTaskName := taskOutput.GetProducerTask()
	if producerTaskName == "" {
		return nil, cfg.err(fmt.Errorf("producerTaskName is empty"))
	}
	outputParameterKey := taskOutput.GetOutputParameterKey()
	if outputParameterKey == "" {
		return nil, cfg.err(fmt.Errorf("output parameter key is empty"))
	}

	producerTaskName = getTaskNameWithTaskID(producerTaskName, cfg.opts.ParentTask.GetTaskId())
	// For the scenario where 2 tasks are defined within a ParallelFor and 1
	// receives the output of the other we must ensure that the downstream task
	// resolves the approriate output of the iteration it is in. With knowing if
	// we are resolving inputs for a task within a ParallelFor DAG we can add
	// the iteration index to the producerTaskName so that we can resolve the
	// correct iteration of that task.
	producerTaskName = InferIndexedTaskName(producerTaskName, cfg.dag.Execution)
	// Get a list of tasks for the current DAG first. The reason we use
	// getDAGTasks instead of mlmd.GetExecutionsInDAG without the dag filter is
	// because the latter does not handle task name collisions in the map which
	// results in a bunch of unhandled edge cases and test failures.
	glog.V(4).Infof("producerTaskName: %v", producerTaskName)
	glog.V(4).Infof("outputParameterKey: %v", outputParameterKey)
	tasks, err := getSubTasks(cfg.opts.ParentTask, cfg.opts.Run.Tasks, nil)
	if err != nil {
		return nil, cfg.err(err)
	}

	producer, ok := tasks[producerTaskName]
	if !ok {
		return nil, cfg.err(fmt.Errorf("producer task, %v, not in tasks", producerTaskName))
	}
	glog.V(4).Info("producer: ", producer)
	glog.V(4).Infof("tasks: %#v", tasks)
	currentTask := producer
	subTaskName := producerTaskName
	// Continue looping until we reach a sub-task that is NOT a DAG.
	for {
		glog.V(4).Info("currentTask: ", currentTask.GetName())
		// If the current task is a DAG:
		if currentTask.GetType() != apiv2beta1.PipelineTaskDetail_RUNTIME {
			// Since currentTask is a DAG, we need to deserialize its
			// output parameter map so that we can look up its
			// corresponding producer sub-task, reassign currentTask,
			// and iterate through this loop again.
			outputParametersCustomProperty, ok := currentTask.GetExecution().GetCustomProperties()["parameter_producer_task"]
			if !ok {
				return nil, cfg.err(fmt.Errorf("task, %v, does not have a parameter_producer_task custom property", currentTask.TaskName()))
			}
			glog.V(4).Infof("outputParametersCustomProperty: %#v", outputParametersCustomProperty)

			dagOutputParametersMap := make(map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec)
			glog.V(4).Infof("outputParametersCustomProperty: %v", outputParametersCustomProperty.GetStructValue())

			for name, value := range outputParametersCustomProperty.GetStructValue().GetFields() {
				outputSpec := &pipelinespec.DagOutputsSpec_DagOutputParameterSpec{}
				err := protojson.Unmarshal([]byte(value.GetStringValue()), outputSpec)
				if err != nil {
					return nil, err
				}
				dagOutputParametersMap[name] = outputSpec
			}

			glog.V(4).Infof("Deserialized dagOutputParametersMap: %v", dagOutputParametersMap)

			// For this section, if the currentTask we are looking for is within
			// a ParallelFor DAG, this means the actual task that produced the
			// output we need has multiple iterations so we have to gather all
			// them and fan them in by collecting them into a list i.e.
			// kfp.dsl.Collected support.
			parentDAG, err := cfg.mlmd.GetExecution(cfg.ctx, currentTask.GetExecution().GetCustomProperties()["parent_dag_id"].GetIntValue())
			if err != nil {
				return nil, cfg.err(err)
			}
			iterations := getParallelForIterationCount(currentTask, parentDAG)
			if iterations > 0 {
				parameterList, _, err := CollectInputs(cfg, subTaskName, tasks, outputParameterKey, false)
				if err != nil {
					return nil, cfg.err(err)
				}
				return parameterList, nil
			}
			// Support for the 2 DagOutputParameterSpec types:
			// ValueFromParameter & ValueFromOneof
			subTaskName, outputParameterKey, err = GetProducerTask(currentTask, tasks, subTaskName, outputParameterKey, false)
			if err != nil {
				return nil, cfg.err(err)
			}
			glog.V(4).Infof("SubTaskName from outputParams: %v", subTaskName)
			glog.V(4).Infof("OutputParameterKey from outputParams: %v", outputParameterKey)
			if subTaskName == "" {
				return nil, cfg.err(fmt.Errorf("producer_subtask not in outputParams"))
			}

			// If the sub-task is a DAG, reassign currentTask and run
			glog.V(4).Infof(
				"Overriding currentTask, %v, output with currentTask's producer_subtask, %v, output.",
				currentTask.TaskName(),
				subTaskName,
			)
			currentTask, ok = tasks[subTaskName]
			if !ok {
				return nil, cfg.err(fmt.Errorf("subTaskName, %v, not in tasks", subTaskName))
			}

		} else {
			_, outputParametersCustomProperty, err := currentTask.GetParameters()
			if err != nil {
				return nil, err
			}
			// Base case
			return outputParametersCustomProperty[outputParameterKey], nil
		}
	}
}

// resolveUpstreamArtifacts resolves input artifacts that come from upstream
// tasks. These tasks can be components/containers, which is relatively
// straightforward, or DAGs, in which case, we need to traverse the graph until
// we arrive at a component/container (since there can be n nested DAGs).
func resolveUpstreamArtifacts(cfg resolveUpstreamOutputsConfig) (*pipelinespec.ArtifactList, error) {
	glog.V(4).Infof("artifactSpec: %#v", cfg.artifactSpec)
	taskOutput := cfg.artifactSpec.GetTaskOutputArtifact()
	glog.V(4).Info("taskOutput: ", taskOutput)
	producerTaskName := taskOutput.GetProducerTask()
	if taskOutput.GetProducerTask() == "" {
		return nil, cfg.err(fmt.Errorf("producer task is empty"))
	}
	if taskOutput.GetOutputArtifactKey() == "" {
		cfg.err(fmt.Errorf("output artifact key is empty"))
	}
	producerTaskName = metadata.GetTaskNameWithDagID(producerTaskName, cfg.dag.Execution.GetID())
	// The main difference between the root ParallelFor DAG and its iteration
	// DAGs is that the root contains the custom property "iteration_count"
	// while the iterations contain "iteration_index". We can use this to
	// determine if we are in a ParallelFor DAG or not. The iteration DAGs will
	// contain the "iteration_index" which is used to resolve the correct output
	// artifact for the downstream task within the iteration. ParallelFor
	// iterations are DAGs themselves, we can verify if we are in a iteration by
	// confirming that the "iteration_index" exists for the DAG of the current
	// task we are attempting to resolve. If the dag contains the
	// "iteration_index", the producerTaskName will be updated appropriately
	producerTaskName = InferIndexedTaskName(producerTaskName, cfg.dag.Execution)
	glog.V(4).Infof("producerTaskName: %v", producerTaskName)
	tasks, err := getDAGTasks(cfg.ctx, cfg.opts, nil)
	if err != nil {
		return nil, cfg.err(err)
	}

	producer, ok := tasks[producerTaskName]
	if !ok {
		return nil, cfg.err(
			fmt.Errorf("cannot find producer task %q", producerTaskName),
		)
	}
	glog.V(4).Info("producer: ", producer)
	glog.V(4).Infof("tasks: %#v", tasks)
	currentTask := producer
	outputArtifactKey := taskOutput.GetOutputArtifactKey()
	subTaskName := producerTaskName
	// Continue looping until we reach a sub-task that is either a ParallelFor
	// task or a Container task.
	for {
		glog.V(4).Info("currentTask: ", currentTask.TaskName())
		// If the current task is a DAG:
		if *currentTask.GetExecution().Type == "system.DAGExecution" {
			// Get the sub-task.
			parentDAG, err := cfg.mlmd.GetExecution(cfg.ctx, currentTask.GetExecution().GetCustomProperties()["parent_dag_id"].GetIntValue())
			if err != nil {
				return nil, cfg.err(err)
			}
			iterations := getParallelForIterationCount(currentTask, parentDAG)
			if iterations > 0 {
				_, artifactList, err := CollectInputs(cfg, subTaskName, tasks, outputArtifactKey, true)
				if err != nil {
					return nil, cfg.err(err)
				}
				return artifactList, nil
			}
			subTaskName, outputArtifactKey, err = GetProducerTask(currentTask, tasks, subTaskName, outputArtifactKey, true)
			if err != nil {
				return nil, cfg.err(err)
			}
			glog.V(4).Infof("ProducerSubtask: %v", subTaskName)
			glog.V(4).Infof("OutputArtifactKey: %v", outputArtifactKey)
			// If the sub-task is a DAG, reassign currentTask and run
			glog.V(4).Infof("currentTask ID: %v", currentTask.GetID())
			glog.V(4).Infof(
				"Overriding currentTask, %v, output with currentTask's producer_subtask, %v, output.",
				currentTask.TaskName(),
				subTaskName,
			)
			currentTask, ok = tasks[subTaskName]
			if !ok {
				return nil, cfg.err(fmt.Errorf("subTaskName, %v, not in tasks", subTaskName))
			}
		} else {
			// Base case, currentTask is a container, not a DAG.
			outputs, err := cfg.mlmd.GetOutputArtifactsByExecutionId(cfg.ctx, currentTask.GetID())
			if err != nil {
				return nil, cfg.err(err)
			}
			glog.V(4).Infof("outputs: %#v", outputs)
			artifact, ok := outputs[outputArtifactKey]
			if !ok {
				cfg.err(
					fmt.Errorf(
						"cannot find output artifact key %q in producer task %q",
						taskOutput.GetOutputArtifactKey(),
						taskOutput.GetProducerTask(),
					),
				)
			}
			runtimeArtifact, err := artifact.ToRuntimeArtifact()
			if err != nil {
				cfg.err(err)
			}
			// Base case
			return &pipelinespec.ArtifactList{
				Artifacts: []*pipelinespec.RuntimeArtifact{runtimeArtifact},
			}, nil
		}
	}
}

// resolvePodSpecInputRuntimeParameter resolves runtime value that is intended to be
// utilized within the Pod Spec. parameterValue takes the form of:
// "{{$.inputs.parameters['pipelinechannel--someParameterName']}}"
//
// parameterValue is a runtime parameter value that has been resolved and included within
// the executor input. Since the pod spec patch cannot dynamically update the underlying
// container template's inputs in an Argo Workflow, this is a workaround for resolving
// such parameters.
//
// If parameter value is not a parameter channel, then a constant value is assumed and
// returned as is.
func resolvePodSpecInputRuntimeParameter(parameterValue string, executorInput *pipelinespec.ExecutorInput) (string, error) {
	if isInputParameterChannel(parameterValue) {
		inputImage, err := extractInputParameterFromChannel(parameterValue)
		if err != nil {
			return "", err
		}
		if val, ok := executorInput.Inputs.ParameterValues[inputImage]; ok {
			return val.GetStringValue(), nil
		} else {
			return "", fmt.Errorf("executorInput did not contain container Image input parameter")
		}
	}
	return parameterValue, nil
}

// resolveK8sJsonParameter resolves a k8s JSON and unmarshal it
// to the provided k8s resource.
//
// Parameters:
//   - pipelineInputParamSpec: An input parameter spec that resolve to a valid JSON
//   - inputParams: InputParams that contain resolution context for pipelineInputParamSpec
//   - res: The k8s resource to unmarshal the json to
func resolveK8sJsonParameter[k8sResource any](
	ctx context.Context,
	opts Options,
	pipelineInputParamSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams map[string]*structpb.Value,
	res *k8sResource,
) error {
	resolvedParam, err := resolveInputParameter(ctx, opts, pipelineInputParamSpec, inputParams)
	if err != nil {
		return fmt.Errorf("failed to resolve k8s parameter: %w", err)
	}
	paramJSON, err := resolvedParam.GetStructValue().MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(paramJSON, &res)
	if err != nil {
		return fmt.Errorf("failed to unmarshal k8s Resource json "+
			"ensure that k8s Resource json correctly adheres to its respective k8s spec: %w", err)
	}
	return nil
}

// CollectInputs performs artifact/parameter collection across a DAG/tree
// using a breadth first search traversal.
func CollectInputs(
	cfg resolveUpstreamOutputsConfig,
	parallelForDAGTaskName string,
	tasks map[string]*metadata.Execution,
	outputKey string,
	isArtifact bool,
) (outputParameterList *structpb.Value, outputArtifactList *pipelinespec.ArtifactList, err error) {
	glog.V(4).Infof("currentTask is a ParallelFor DAG. Attempting to gather all nested producer_subtasks")
	// Set some helpers for the start and looping for BFS
	var currentTask *metadata.Execution
	var workingSubTaskName string
	workingOutputKey := outputKey
	previousWorkingOutputKey := outputKey
	// Instantiate the lists values that will hold all values pulled from the
	// tasks of each iteration.
	parallelForParameterList := make([]*structpb.Value, 0)
	parallelForArtifactList := make([]*pipelinespec.RuntimeArtifact, 0)
	tasksToResolve := make([]string, 0)
	// Set up the queue for BFS by setting the parallelFor DAG task as the
	// initial node. The loop will add the iteration dag task names for us into
	// the slice/queue.
	tasksToResolve = append(tasksToResolve, parallelForDAGTaskName)
	previousTaskName := tasks[tasksToResolve[0]].TaskName()

	for len(tasksToResolve) > 0 {
		// The starterQueue contains the first set of child DAGs from the
		// parallelFor, i.e. the iteration dags.
		glog.V(4).Infof("tasksToResolve: %v", tasksToResolve)
		currentTaskName := tasksToResolve[0]
		tasksToResolve = tasksToResolve[1:]

		currentTask = tasks[currentTaskName]

		// We check if these values need to be updated going through the
		// resolution of dags/tasks Most commonly the subTaskName will change
		// for both parameter & artifact resolution. For parameter resolutions,
		// the outputParameterKey can change, and is used for extracting the
		// appropriate field off of the struct set for the outputs on the task
		// in question.

		// An issue arises if we update the outputParameterKey but there exists
		// multiple iterations of the same task and we haven't fully parsed all
		// iterations. We will encounter a scenario where we will attempt to
		// extract fields from the struct with the wrong key. Hence, the
		// condition below. NOTE: This is only an issue for Parameter resolution
		// and does not interfere with Artifact resolution.
		if currentTask.TaskName() == previousTaskName {
			workingOutputKey = previousWorkingOutputKey
		}

		previousTaskName = currentTask.TaskName()
		previousWorkingOutputKey = workingOutputKey
		workingSubTaskName, workingOutputKey, _ = GetProducerTask(currentTask, tasks, workingSubTaskName, workingOutputKey, isArtifact)

		glog.V(4).Infof("currentTask ID: %v", currentTask.GetID())
		glog.V(4).Infof("currentTask Name: %v", currentTask.TaskName())
		glog.V(4).Infof("currentTask Type: %v", currentTask.GetExecution().GetType())
		glog.V(4).Infof("workingSubTaskName %v", workingSubTaskName)
		glog.V(4).Infof("workingOutputKey: %v", workingOutputKey)

		iterations := currentTask.GetExecution().GetCustomProperties()["iteration_count"]
		iterationIndex := currentTask.GetExecution().GetCustomProperties()["iteration_index"]

		// Base cases for handling the task that actually maps to the task that
		// created the artifact/parameter we are searching for.

		//  Base case 1: currentTask is a ContainerExecution that we can load
		//  the values off of.
		if *currentTask.GetExecution().Type == "system.ContainerExecution" {
			glog.V(4).Infof("currentTask, %v, is a ContainerExecution", currentTaskName)
			paramValue, artifact, err := collectContainerOutput(cfg, currentTask, workingOutputKey, isArtifact)
			if err != nil {
				return nil, nil, err
			}
			if isArtifact {
				parallelForArtifactList = append(parallelForArtifactList, artifact)
				glog.V(4).Infof("parallelForArtifactList: %v", parallelForArtifactList)
			} else {
				parallelForParameterList = append(parallelForParameterList, paramValue)
				glog.V(4).Infof("parallelForParameterList: %v", parallelForParameterList)
			}
			continue
		}
		// Base case 2: currentTask is a DAGExecution within a loop but is
		// NOT a ParallelFor Head DAG
		if iterations == nil {
			tempSubTaskName := workingSubTaskName
			if iterationIndex != nil {
				// handle for parallel iteration dag, i.e one of the DAG
				// instances of the loop.
				tempSubTaskName = metadata.GetParallelForTaskName(tempSubTaskName, iterationIndex.GetIntValue())
				glog.V(4).Infof("subTaskIterationName: %v", tempSubTaskName)
			}
			glog.V(4).Infof("tempSubTaskName: %v", tempSubTaskName)
			tasksToResolve = append(tasksToResolve, tempSubTaskName)
			continue
		}

		// If the currentTask is not a ContainerExecution AND we have the
		// custom property set for "iteration_count", we can deduce that
		// currentTask is in fact a ParallelFor Head DAG, thus we need to add
		// its iteration DAGs to the queue.

		for i := range iterations.GetIntValue() {
			loopName := metadata.GetTaskNameWithDagID(currentTask.TaskName(), currentTask.GetID())
			loopIterationName := metadata.GetParallelForTaskName(loopName, i)
			glog.V(4).Infof("loopIterationName: %v", loopIterationName)
			tasksToResolve = append(tasksToResolve, loopIterationName)
		}
	}

	outputParameterList = &structpb.Value{
		Kind: &structpb.Value_ListValue{
			ListValue: &structpb.ListValue{
				Values: parallelForParameterList,
			},
		},
	}
	outputArtifactList = &pipelinespec.ArtifactList{
		Artifacts: parallelForArtifactList,
	}
	glog.V(4).Infof("outputParameterList: %#v", outputParameterList)
	glog.V(4).Infof("outputArtifactList: %#v", outputArtifactList)
	return outputParameterList, outputArtifactList, nil
}

// collectContainerOutput pulls either the artifact or parameter that is a
// task's output where said task was called within a parallelFor loop
func collectContainerOutput(
	cfg resolveUpstreamOutputsConfig,
	currentTask *metadata.Execution,
	workingOutputKey string,
	isArtifact bool,
) (*structpb.Value, *pipelinespec.RuntimeArtifact, error) {
	var param *structpb.Value
	var artifact *pipelinespec.RuntimeArtifact
	if isArtifact {
		outputArtifacts, err := cfg.mlmd.GetOutputArtifactsByExecutionId(cfg.ctx, currentTask.GetID())
		if err != nil {
			return nil, nil, err
		}
		glog.V(4).Infof("outputArtifacts: %#v", outputArtifacts)
		glog.V(4).Infof("outputKey: %v", workingOutputKey)
		artifact, err = outputArtifacts[workingOutputKey].ToRuntimeArtifact()
		if err != nil {
			return nil, nil, cfg.err(err)
		}
		glog.V(4).Infof("runtimeArtifact: %v", artifact)
	} else {
		_, outputParameters, err := currentTask.GetParameters()
		glog.V(4).Infof("outputParameters: %v", outputParameters)
		if err != nil {
			return nil, nil, cfg.err(err)
		}
		param = outputParameters[workingOutputKey]
	}
	return param, artifact, nil
}

// GetProducerTask gets the updated ProducerSubTask /
// Output[Artifact|Parameter]Key if they exists, else it returns the original
// input.
func GetProducerTask(parentTask *metadata.Execution, tasks map[string]*metadata.Execution, subTaskName string, outputKey string, isArtifact bool) (producerSubTaskName string, tempOutputKey string, err error) {
	tempOutputKey = outputKey
	if isArtifact {
		producerTaskValue := parentTask.GetExecution().GetCustomProperties()["artifact_producer_task"]
		if producerTaskValue != nil {
			var tempOutputArtifacts map[string]*pipelinespec.DagOutputsSpec_DagOutputArtifactSpec
			err := json.Unmarshal([]byte(producerTaskValue.GetStringValue()), &tempOutputArtifacts)
			if err != nil {
				return "", "", err
			}
			glog.V(4).Infof("tempOutputsArtifacts: %v", tempOutputArtifacts)
			glog.V(4).Infof("outputArtifactKey: %v", outputKey)
			tempSelectors := tempOutputArtifacts[outputKey].GetArtifactSelectors()
			if len(tempSelectors) > 0 {
				producerSubTaskName = tempSelectors[len(tempSelectors)-1].ProducerSubtask
				tempOutputKey = tempSelectors[len(tempSelectors)-1].OutputArtifactKey
			}
		}

	} else {
		producerTaskValue := parentTask.GetExecution().GetCustomProperties()["parameter_producer_task"]
		if producerTaskValue != nil {
			tempOutputParametersMap := make(map[string]*pipelinespec.DagOutputsSpec_DagOutputParameterSpec)
			for name, value := range producerTaskValue.GetStructValue().GetFields() {
				outputSpec := &pipelinespec.DagOutputsSpec_DagOutputParameterSpec{}
				err := protojson.Unmarshal([]byte(value.GetStringValue()), outputSpec)
				if err != nil {
					return "", "", err
				}
				tempOutputParametersMap[name] = outputSpec
			}
			glog.V(4).Infof("tempOutputParametersMap: %#v", tempOutputParametersMap)
			switch tempOutputParametersMap[tempOutputKey].Kind.(type) {
			case *pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromParameter:
				producerSubTaskName = tempOutputParametersMap[tempOutputKey].GetValueFromParameter().GetProducerSubtask()
				tempOutputKey = tempOutputParametersMap[tempOutputKey].GetValueFromParameter().GetOutputParameterKey()
			case *pipelinespec.DagOutputsSpec_DagOutputParameterSpec_ValueFromOneof:
				// When OneOf is specified in a pipeline, the output of only 1
				// task is consumed even though there may be more than 1 task
				// output set. In this case we will attempt to grab the first
				// successful task output.
				paramSelectors := tempOutputParametersMap[tempOutputKey].GetValueFromOneof().GetParameterSelectors()
				glog.V(4).Infof("paramSelectors: %v", paramSelectors)
				// Since we have the tasks map, we can iterate through the
				// parameterSelectors if the ProducerSubTask is not present in
				// the task map and then assign the new OutputParameterKey only
				// if it exists.
				successfulOneOfTask := false
				for _, paramSelector := range paramSelectors {
					producerSubTaskName = paramSelector.GetProducerSubtask()
					// Used just for retrieval since we lookup the task in the map
					updatedSubTaskName := metadata.GetTaskNameWithDagID(producerSubTaskName, parentTask.GetID())
					glog.V(4).Infof("subTaskName with Dag ID from paramSelector: %v", updatedSubTaskName)
					glog.V(4).Infof("outputParameterKey from paramSelector: %v", paramSelector.GetOutputParameterKey())
					if subTask, ok := tasks[updatedSubTaskName]; ok {
						subTaskState := subTask.GetExecution().GetLastKnownState().String()
						glog.V(4).Infof("subTask: %w , subTaskState: %v", updatedSubTaskName, subTaskState)
						if subTaskState == "CACHED" || subTaskState == "COMPLETE" {
							tempOutputKey = paramSelector.GetOutputParameterKey()
							successfulOneOfTask = true
							break
						}
					}
				}
				if !successfulOneOfTask {
					return "", "", fmt.Errorf("processing OneOf: No successful task found")
				}
			}
		}
	}
	if producerSubTaskName != "" {
		producerSubTaskName = metadata.GetTaskNameWithDagID(producerSubTaskName, parentTask.GetID())
	} else {
		producerSubTaskName = subTaskName
	}
	return producerSubTaskName, tempOutputKey, nil
}

func getParallelForTaskName(taskName string, iterationIndex int64) string {
	return fmt.Sprintf("%s_idx_%d", taskName, iterationIndex)
}

// Helper for determining if the current producerTask in question needs to pull from an iteration dag that it may exist in.
func InferIndexedTaskName(producerTaskName string, task *apiv2beta1.PipelineTaskDetail) string {
	// Check if the Task in question is a parallelFor iteration Task. If it is, we need to
	// update the producerTaskName so the downstream task resolves the appropriate index.
	if task.GetType() == apiv2beta1.PipelineTaskDetail_LOOP_ITERATION {
		taskIterationIndex := task.GetTypeAttributes().GetIterationIndex()
		producerTaskName = getParallelForTaskName(producerTaskName, taskIterationIndex)
		glog.V(4).Infof("TaskIteration - ProducerTaskName: %v", producerTaskName)
		glog.Infof("Attempting to retrieve outputs from a ParallelFor iteration")
	}
	return producerTaskName
}
