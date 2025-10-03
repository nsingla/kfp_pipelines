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
