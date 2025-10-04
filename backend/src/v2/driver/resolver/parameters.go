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
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"google.golang.org/protobuf/types/known/structpb"
)

func resolveParameters(opts common.Options) ([]ParameterMetadata, *int, error) {
	var parameters []ParameterMetadata
	for name, paramSpec := range opts.Task.GetInputs().GetParameters() {
		if compParam := opts.Component.GetInputDefinitions().GetParameters()[name]; compParam != nil {
			// Skip resolving dsl.TaskConfig because that information is only available after initPodSpecPatch and
			// extendPodSpecPatch are called.
			if compParam.GetParameterType() == pipelinespec.ParameterType_TASK_CONFIG {
				continue
			}
		}

		v, err := resolveInputParameter(opts, paramSpec, opts.ParentTask.Inputs.GetParameters())
		if err != nil {
			if !errors.Is(err, ErrResolvedInputNull) {
				return nil, nil, err
			}
			componentParam, ok := opts.Component.GetInputDefinitions().GetParameters()[name]
			if !ok {
				return nil, nil, fmt.Errorf("parameter %s not found in component input definitions", name)
			}
			// If the resolved parameter was null and the component input parameter is optional, just skip setting
			// it and the launcher will handle defaults.
			if componentParam != nil && componentParam.IsOptional {
				continue
			}
			return nil, nil, err
		}
		parameters = append(parameters, ParameterMetadata{
			Key:                name,
			ParameterIO:        v,
			InputParameterSpec: paramSpec,
		})
	}

	// Handle Parameter Iterator
	var iterationCount *int
	var value *structpb.Value
	if opts.Task.GetParameterIterator() != nil {
		iterator := opts.Task.GetParameterIterator()
		switch iterator.GetItems().GetKind().(type) {
		case *pipelinespec.ParameterIteratorSpec_ItemsSpec_InputParameter:
			// This should be the key input into the for loop task
			iteratorInputDefinitionKey := iterator.GetItemInput()
			// Used to look up the Parameter from the resolved list
			// The key here should map to a ParameterMetadata.Key that
			// was resolved in the prior loop.
			sourceInputParameterKey := iterator.GetItems().GetInputParameter()

			var err error
			parameterIO, err := findParameterByIOKey(sourceInputParameterKey, parameters)
			if err != nil {
				return nil, nil, err
			}
			value = parameterIO.GetValue()
			parameters = append(parameters, ParameterMetadata{
				Key: iteratorInputDefinitionKey,
				ParameterIO: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
					Value:    value,
					Type:     apiv2beta1.IOType_ITERATOR_INPUT,
					Producer: parameterIO.Producer,
				},
				ParameterIterator: iterator,
			})
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
			parameters = append(parameters, ParameterMetadata{
				Key: iterator.GetItemInput(),
				ParameterIO: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
					Value:    value,
					Type:     apiv2beta1.IOType_ITERATOR_INPUT_RAW,
					Producer: nil, // Raw Inputs have no producer
				},
				ParameterIterator: iterator,
			})
		default:
			return nil, nil, fmt.Errorf("cannot find parameter iterator")
		}
		items, err := getItems(value)
		if err != nil {
			return nil, nil, err
		}
		count := len(items)
		iterationCount = &count
	}

	return parameters, iterationCount, nil
}

func resolveInputParameter(
	opts common.Options,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter,
) (*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter, error) {

	switch t := paramSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter:
		glog.V(4).Infof("resolving component input parameter %s", paramSpec.GetComponentInputParameter())
		return resolveParameterComponentInputParameter(opts, paramSpec, inputParams)
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter: // TODO(HumairAK)
		glog.V(4).Infof("resolving task output parameter %s", paramSpec.GetTaskOutputParameter().String())
		return nil, paramError(paramSpec, fmt.Errorf("task output parameter not supported yet"))
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
				return ioParameter, nil
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
					return nil, fmt.Errorf("parent task should not be nil")
				}
				v = structpb.NewStringValue(fmt.Sprintf("%s", opts.ParentTask.GetTaskId()))
			default:
				v = val
			}
			ioParameter := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
				ParameterKey: "",
				Value:        v,
			}
			return ioParameter, nil
		default:
			return nil, paramError(paramSpec, fmt.Errorf("param runtime value spec of type %T not implemented", t))
		}
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_: // TODO(HumairAK)
		glog.V(4).Infof("resolving Task Final Statu %s", paramSpec.GetTaskFinalStatus().String())
		return nil, paramError(paramSpec, fmt.Errorf("task output parameter not supported yet"))
	default:
		return nil, paramError(paramSpec, fmt.Errorf("parameter spec of type %T not implemented yet", t))
	}
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
