package resolver

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/component"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"google.golang.org/protobuf/types/known/structpb"
)

func resolveInputParameter(
	ctx context.Context,
	opts common.Options,
	paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec,
	inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter,
) (*structpb.Value, error) {

	switch t := paramSpec.Kind.(type) {
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter:
		glog.V(4).Infof("resolving component input parameter %s", paramSpec.GetComponentInputParameter())
		return resolveParameterComponentInputParameter(opts, paramSpec, inputParams)
	case *pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter:
		// TODO(HumairAK)
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
		// TODO(HumairAK)
		glog.V(4).Infof("resolving Task Final Statu %s", paramSpec.GetTaskFinalStatus().String())
		return nil, paramError(paramSpec, fmt.Errorf("task output parameter not supported yet"))
	default:
		return nil, paramError(paramSpec, fmt.Errorf("parameter spec of type %T not implemented yet", t))
	}
}

func resolveParameterComponentInputParameter(opts common.Options, paramSpec *pipelinespec.TaskInputsSpec_InputParameterSpec, inputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter) (*structpb.Value, error) {
	paramName := paramSpec.GetComponentInputParameter()
	if paramName == "" {
		return nil, paramError(paramSpec, fmt.Errorf("empty component input"))
	}
	isPipelineChanel := common.IsPipelineChannel(paramName)
	for _, param := range inputParams {
		generateName, err := common.IOFieldsToPipelineChannelName(param.GetParameterName(), param.GetProducer(), isPipelineChanel)
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
			if common.IsLoopArgument(paramName) {
				if _, ok := value.GetKind().(*structpb.Value_ListValue); !ok {
					return nil, fmt.Errorf("loop argument %s must be a list value", paramName)
				}
				listValues := value.GetListValue().GetValues()
				if len(listValues) == 0 {
					return nil, fmt.Errorf("loop argument %s must have at least one value", paramName)
				}
				if len(listValues) <= opts.IterationIndex+1 {
					return nil, fmt.Errorf(
						"loop argument %s has only %d values,"+
							" but index %d is requested, which is out "+
							"of bounds", paramName, len(listValues), opts.IterationIndex)
				}
				return listValues[opts.IterationIndex], nil
			}
			return value, nil
		}
	}
	return nil, fmt.Errorf("failed to find input param %s", paramName)
}
