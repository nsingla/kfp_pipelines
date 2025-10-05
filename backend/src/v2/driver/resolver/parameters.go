package resolver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"google.golang.org/protobuf/types/known/structpb"
)

func resolveParameters(opts common.Options) ([]ParameterMetadata, error) {
	var parameters []ParameterMetadata
	for key, paramSpec := range opts.Task.GetInputs().GetParameters() {
		if compParam := opts.Component.GetInputDefinitions().GetParameters()[key]; compParam != nil {
			// Skip resolving dsl.TaskConfig because that information is only available after initPodSpecPatch and
			// extendPodSpecPatch are called.
			if compParam.GetParameterType() == pipelinespec.ParameterType_TASK_CONFIG {
				continue
			}
		}

		v, ioType, err := resolveInputParameter(opts, paramSpec, opts.ParentTask.Inputs.GetParameters())
		if err != nil {
			if !errors.Is(err, ErrResolvedInputNull) {
				return nil, err
			}
			componentParam, ok := opts.Component.GetInputDefinitions().GetParameters()[key]
			if !ok {
				return nil, fmt.Errorf("parameter %s not found in component input definitions", key)
			}
			// If the resolved parameter was null and the component input parameter is optional, just skip setting
			// it and the launcher will handle defaults.
			if componentParam != nil && componentParam.IsOptional {
				continue
			}
			return nil, err
		}
		pm := ParameterMetadata{
			Key:                key,
			InputParameterSpec: paramSpec,
			ParameterIO: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				Value:        v.GetValue(),
				Type:         ioType,
				ParameterKey: key,
				Producer:     &apiv2beta1.IOProducer{TaskName: opts.TaskName},
			},
		}
		if opts.IterationIndex >= 0 {
			pm.ParameterIO.Producer.Iteration = util.Int64Pointer(int64(opts.IterationIndex))
		}
		parameters = append(parameters, pm)
	}

	return parameters, nil
}

func resolveInputParameter(
	opts common.Options,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, apiv2beta1.IOType, error) {
	switch t := paramSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter:
		glog.V(4).Infof("resolving component input parameter %s", paramSpec.GetComponentInputParameter())
		resolvedInput, err := resolveParameterComponentInputParameter(opts, paramSpec, inputParams)
		if err != nil {
			return nil, apiv2beta1.IOType_COMPONENT_INPUT, err
		}
		return resolvedInput, apiv2beta1.IOType_COMPONENT_INPUT, nil
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:

		parameter, err := resolveUpstreamParameters(opts, paramSpec)
		if err != nil {
			return nil, apiv2beta1.IOType_TASK_OUTPUT_INPUT, err
		}
		ioType := apiv2beta1.IOType_TASK_OUTPUT_INPUT
		if parameter.GetType() == apiv2beta1.IOType_COLLECTED_INPUTS {
			ioType = apiv2beta1.IOType_COLLECTED_INPUTS
		}
		return parameter, ioType, nil
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue:
		glog.V(4).Infof("resolving runtime value %s", paramSpec.GetRuntimeValue().String())
		runtimeValue := paramSpec.GetRuntimeValue()
		switch t := runtimeValue.Value.(type) {
		case *pipelinespec.ValueOrRuntimeParameter_Constant:
			val := runtimeValue.GetConstant()
			valStr := val.GetStringValue()
			var v *structpb.Value
			if strings.Contains(valStr, "{{$.workspace_path}}") {
				v = structpb.NewStringValue(strings.ReplaceAll(valStr, "{{$.workspace_path}}", component.WorkspaceMountPath))
				ioParameter := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
					ParameterKey: "",
					Value:        v,
				}
				return ioParameter, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, nil
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
				if opts.ParentTask == nil {
					return nil, apiv2beta1.IOType_UNSPECIFIED, fmt.Errorf("parent task should not be nil")
				}
				v = structpb.NewStringValue(fmt.Sprintf("%s", opts.ParentTask.GetTaskId()))
			default:
				v = val
			}
			ioParameter := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				ParameterKey: "",
				Value:        v,
			}
			return ioParameter, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, nil
		default:
			return nil, apiv2beta1.IOType_UNSPECIFIED, paramError(paramSpec, fmt.Errorf("param runtime value spec of type %T not implemented", t))
		}
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_: // TODO(HumairAK)
		glog.V(4).Infof("resolving Task Final Statu %s", paramSpec.GetTaskFinalStatus().String())
		return nil, apiv2beta1.IOType_UNSPECIFIED, paramError(paramSpec, fmt.Errorf("task output parameter not supported yet"))
	default:
		return nil, apiv2beta1.IOType_UNSPECIFIED, paramError(paramSpec, fmt.Errorf("parameter spec of type %T not implemented yet", t))
	}
}

func resolveUpstreamParameters(
	opts common.Options,
	spec *pipelinespec.TaskInputsSpec_InputParameterSpec,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, error) {
	tasks, err := getSubTasks(opts.ParentTask, opts.Run.Tasks, nil)
	if err != nil {
		return nil, err
	}
	if tasks == nil {
		return nil, fmt.Errorf("failed to get sub tasks for task %s", opts.ParentTask.Name)
	}
	producerTaskAmbiguousName := spec.GetTaskOutputParameter().GetProducerTask()
	if producerTaskAmbiguousName == "" {
		return nil, fmt.Errorf("producerTask task cannot be empty")
	}
	producerTaskUniqueName := getTaskNameWithTaskID(producerTaskAmbiguousName, opts.ParentTask.GetTaskId())
	if opts.IterationIndex >= 0 {
		producerTaskUniqueName = getTaskNameWithIterationIndex(producerTaskUniqueName, int64(opts.IterationIndex))
	}
	// producerTask is the specific task guaranteed to have the output parameter
	// producerTaskUniqueName may look something like "task_name_a_dag_id_1_idx_0"
	producerTask := tasks[producerTaskUniqueName]
	if producerTask == nil {
		return nil, fmt.Errorf("producerTask task %s not found", producerTaskUniqueName)
	}
	outputKey := spec.GetTaskOutputParameter().GetOutputParameterKey()
	outputs := producerTask.GetOutputs().GetParameters()
	outputIO, err := findParameterByProducerKeyInList(outputKey, outputs)
	if err != nil {
		return nil, err
	}
	if outputIO == nil {
		return nil, fmt.Errorf("output parameter %s not found", outputKey)
	}
	return outputIO, nil
}

func resolveParameterComponentInputParameter(
	opts common.Options,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, error) {
	paramName := paramSpec.GetComponentInputParameter()
	if paramName == "" {
		return nil, paramError(paramSpec, fmt.Errorf("empty component input"))
	}
	for _, param := range inputParams {
		generateName := param.ParameterKey
		if paramName == generateName {
			if !common.IsLoopArgument(paramName) {
				return param, nil
			}
			// If the input is a loop argument, we need to check if the iteration index matches the current iteration.
			if param.Producer != nil && param.Producer.Iteration != nil && *param.Producer.Iteration == int64(opts.IterationIndex) {
				return param, nil
			}
		}
	}
	return nil, fmt.Errorf("failed to find input param %s", paramName)
}

// resolveParameterIterator handles parameter Iterator Input resolution
func resolveParameterIterator(
	opts common.Options,
	parameters []ParameterMetadata,
) ([]ParameterMetadata, *int, error) {
	var value *structpb.Value
	var iteratorInputDefinitionKey string
	var iterator *pipelinespec.ParameterIteratorSpec
	iterator = opts.Task.GetParameterIterator()
	switch iterator.GetItems().GetKind().(type) {
	case *pipelinespec.ParameterIteratorSpec_ItemsSpec_InputParameter:
		// This should be the key input into the for loop task
		iteratorInputDefinitionKey = iterator.GetItemInput()
		// Used to look up the Parameter from the resolved list
		// The key here should map to a ParameterMetadata.Key that
		// was resolved in the prior loop.
		sourceInputParameterKey := iterator.GetItems().GetInputParameter()

		// Determine if the parameter is a parameter or an artifact

		var err error
		parameterIO, err := findParameterByIOKey(sourceInputParameterKey, parameters)
		if err != nil {
			return nil, nil, err
		}
		value = parameterIO.GetValue()
	case *pipelinespec.ParameterIteratorSpec_ItemsSpec_Raw:
		valueRaw := iterator.GetItems().GetRaw()
		var unmarshalledRaw interface{}
		err := json.Unmarshal([]byte(valueRaw), &unmarshalledRaw)
		if err != nil {
			return nil, nil, fmt.Errorf("error unmarshall raw string: %q", err)
		}
		value, err = structpb.NewValue(unmarshalledRaw)
		if err != nil {
			return nil, nil, fmt.Errorf("error converting unmarshalled raw string into protobuf Value type: %q", err)
		}
		iteratorInputDefinitionKey = iterator.GetItemInput()

	default:
		return nil, nil, fmt.Errorf("cannot find parameter iterator")
	}

	items, err := getItems(value)
	if err != nil {
		return nil, nil, err
	}

	var parameterMetadataList []ParameterMetadata
	for i, item := range items {
		pm := ParameterMetadata{
			Key: iteratorInputDefinitionKey,
			ParameterIO: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				Value:        item,
				Type:         apiv2beta1.IOType_ITERATOR_INPUT,
				ParameterKey: iteratorInputDefinitionKey,
				Producer: &apiv2beta1.IOProducer{
					TaskName:  opts.TaskName,
					Iteration: util.Int64Pointer(int64(i)),
				},
			},
			ParameterIterator: iterator,
		}
		parameterMetadataList = append(parameterMetadataList, pm)
	}
	count := len(items)
	return parameterMetadataList, &count, nil
}

// getItems iteration items from a structpb.Value.
// Return value may be
// * a list of JSON serializable structs
// * a list of structpb.Value
func getItems(value *structpb.Value) (items []*structpb.Value, err error) {
	switch v := value.GetKind().(type) {
	case *structpb.Value_ListValue:
		return v.ListValue.GetValues(), nil
	case *structpb.Value_StringValue:
		listValue := structpb.Value{}
		if err = listValue.UnmarshalJSON([]byte(v.StringValue)); err != nil {
			return nil, err
		}
		return listValue.GetListValue().GetValues(), nil
	default:
		return nil, fmt.Errorf("value of type %T cannot be iterated", v)
	}
}

// Convert []*structpb.Value to a *structpb.Value_ListValue
func toListValue(items []*structpb.Value) *structpb.Value {
	listValue := structpb.Value{}
	listValue.Kind = &structpb.Value_ListValue{
		ListValue: &structpb.ListValue{
			Values: items,
		},
	}
	return &listValue
}

func ResolvePodSpecInputRuntimeParameter(n string, input *pipelinespec.ExecutorInput) (string, error) {
	return "", nil
}

func ResolveK8sJsonParameter[k8sResource any](
	ctx context.Context,
	opts common.Options,
	selectorJson *pipelinespec.TaskInputsSpec_InputParameterSpec,
	params []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, res *k8sResource) error {

	return nil
}

func ResolveInputParameterStr(
	ctx context.Context,
	opts common.Options,
	parameter *pipelinespec.TaskInputsSpec_InputParameterSpec,
	params []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter) (*structpb.Value, error) {

	return nil, nil
}
func ResolveInputParameter(
	ctx context.Context,
	opts common.Options,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter,
) (*structpb.Value, error) {
	return nil, nil
}

func findParameterByProducerKeyInList(
	producerKey string,
	parametersIO []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, error) {
	var parameterIOList []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter
	for _, parameterIO := range parametersIO {
		if parameterIO.GetParameterKey() == producerKey {
			parameterIOList = append(parameterIOList, parameterIO)
		}
	}
	if len(parameterIOList) == 0 {
		return nil, fmt.Errorf("parameter with producer key %s not found", producerKey)
	}

	ioType := apiv2beta1.IOType_TASK_OUTPUT_INPUT
	// This occurs in the parallelFor case, where multiple iterations resulted in the same
	// producer key.
	isCollection := len(parameterIOList) > 1
	if isCollection {
		var parameterValues []*structpb.Value
		for _, parameterIO := range parameterIOList {
			//  Check correctness by validating the type of all parameters
			if parameterIO.Type != apiv2beta1.IOType_ITERATOR_OUTPUT {
				return nil, fmt.Errorf("encountered a non iterator output that has the same producer key (%s)", producerKey)
			}
			// Support for an iterator over list of parameters is not supported yet.
			parameterValues = append(parameterValues, parameterIO.GetValue())
		}
		ioType = apiv2beta1.IOType_COLLECTED_INPUTS
		newParameterIO := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			Value:        toListValue(parameterValues),
			Type:         ioType,
			ParameterKey: producerKey,
			// This is unused by the caller
			Producer: nil,
		}
		return newParameterIO, nil
	}
	return parameterIOList[0], nil
}
