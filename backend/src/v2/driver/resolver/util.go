package resolver

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/expression"
	"google.golang.org/protobuf/types/known/structpb"
)

func pbValueToText(v *structpb.Value) (string, error) {
	wrap := func(err error) error {
		return fmt.Errorf("failed to convert protobuf.Value to text: %w", err)
	}
	if v == nil {
		return "", nil
	}
	var text string
	switch t := v.Kind.(type) {
	case *structpb.Value_NullValue:
		text = ""
	case *structpb.Value_StringValue:
		text = v.GetStringValue()
	case *structpb.Value_NumberValue:
		text = strconv.FormatFloat(v.GetNumberValue(), 'f', -1, 64)
	case *structpb.Value_BoolValue:
		text = strconv.FormatBool(v.GetBoolValue())
	case *structpb.Value_ListValue:
		b, err := json.Marshal(v.GetListValue())
		if err != nil {
			return "", wrap(fmt.Errorf("failed to JSON-marshal a list: %w", err))
		}
		text = string(b)
	case *structpb.Value_StructValue:
		b, err := json.Marshal(v.GetStructValue())
		if err != nil {
			return "", wrap(fmt.Errorf("failed to JSON-marshal a struct: %w", err))
		}
		text = string(b)
	default:
		return "", wrap(fmt.Errorf("unknown type %T", t))
	}
	return text, nil
}

func fetchTaskInTaskList(taskID string, tasks []*apiV2beta1.PipelineTaskDetail) (*apiV2beta1.PipelineTaskDetail, error) {
	for _, t := range tasks {
		if t.GetTaskId() == taskID {
			return t, nil
		}
	}
	return nil, fmt.Errorf("failed to find task %s", taskID)
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
