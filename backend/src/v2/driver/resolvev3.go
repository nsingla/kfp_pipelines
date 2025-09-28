package driver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"google.golang.org/protobuf/types/known/structpb"
)

var paramError = func(paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec, err error) error {
	return fmt.Errorf("resolving input parameter with spec %s: %w", paramSpec, err)
}

func resolveInputsV3(
	ctx context.Context,
	iterationIndex *int,
	opts Options,
	expr *expression.Expr,
) (*pipelinespec.ExecutorInput_Inputs, error) {
	inputs := &pipelinespec.ExecutorInput_Inputs{
		ParameterValues: make(map[string]*structpb.Value),
		Artifacts:       make(map[string]*pipelinespec.ArtifactList),
	}
	// Handle parameters
	for name, paramSpec := range opts.Task.GetInputs().GetParameters() {
		if compParam := opts.Component.GetInputDefinitions().GetParameters()[name]; compParam != nil {
			// Skip resolving dsl.TaskConfig because that information is only available after initPodSpecPatch and
			// extendPodSpecPatch are called.
			if compParam.GetParameterType() == pipelinespec.ParameterType_TASK_CONFIG {
				continue
			}
		}

		v, err := resolveInputParameterV3(ctx, opts, paramSpec, opts.ParentTask.Inputs.GetParameters())
		if err != nil {
			if !errors.Is(err, ErrResolvedParameterNull) {
				return nil, err
			}
			componentParam, ok := opts.Component.GetInputDefinitions().GetParameters()[name]
			if !ok {
				return nil, fmt.Errorf("parameter %s not found in component input definitions", name)
			}
			// If the resolved parameter was null and the component input parameter is optional, just skip setting
			// it and the launcher will handle defaults.
			if componentParam != nil && componentParam.IsOptional {
				continue
			}
			return nil, err
		}
		inputs.ParameterValues[name] = v
	}

	return inputs, nil
}

func resolveInputParameterV3(
	ctx context.Context,
	opts Options,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter,
) (*structpb.Value, error) {

	switch t := paramSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter:
		glog.V(4).Infof("resolving component input parameter %s", paramSpec.GetComponentInputParameter())
		return resolveParameterComponentInputParameter(paramSpec, inputParams)
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:
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
				if opts.ParentTask == nil {
					return nil, fmt.Errorf("parent task should not be nil")
				}
				v = structpb.NewStringValue(fmt.Sprintf("%s", opts.ParentTask.GetTaskId()))
			default:
				v = val
			}
			return v, nil
		default:
			return nil, paramError(paramSpec, fmt.Errorf("param runtime value spec of type %T not implemented", t))
		}
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskFinalStatus_:
		glog.V(4).Infof("resolving Task Final Statu %s", paramSpec.GetTaskFinalStatus().String())
		return nil, paramError(paramSpec, fmt.Errorf("task output parameter not supported yet"))
	default:
		return nil, paramError(paramSpec, fmt.Errorf("parameter spec of type %T not implemented yet", t))
	}
}

func resolveParameterComponentInputParameter(
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter) (*structpb.Value, error) {

	paramName := paramSpec.GetComponentInputParameter()
	if paramName == "" {
		return nil, paramError(paramSpec, fmt.Errorf("empty component input"))
	}

	for _, param := range inputParams {
		generateName, err := parseIONameOrPipelineChannel(param.GetParameterName(), param.GetProducer())
		if err != nil {
			return nil, err
		}
		if paramName == generateName {
			value := param.GetValue()
			if _, isNullValue := value.GetKind().(*structpb.Value_NullValue); isNullValue {
				// Null values are only allowed for optional pipeline input parameters with no values. The caller has this
				// context to know if this is allowed.
				return nil, fmt.Errorf("%w: %s", ErrResolvedParameterNull, paramName)
			}
			return value, nil
		}
	}
	return nil, fmt.Errorf("failed to find input param %s", paramName)
}

func handleParameterExpressionSelector(
	task *pipelinespec.PipelineTaskSpec,
	inputs *pipelinespec.ExecutorInput_Inputs,
	expr *expression.Expr,
) error {
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

func handleParamTypeValidationAndConversion(
	task *pipelinespec.PipelineTaskSpec,
	inputsSpec *pipelinespec.ComponentInputsSpec,
	inputs *pipelinespec.ExecutorInput_Inputs,
	isIterationDriver bool,
	parametersSetNilByDriver map[string]bool,
) error {
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
				text, err := pbValueToText(value)
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
