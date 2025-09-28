// Copyright 2021-2024 The Kubeflow Authors
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
	"fmt"
	"regexp"
	"strings"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"google.golang.org/protobuf/types/known/structpb"
)

// inputPipelineChannelPattern define a regex pattern to match the content within single quotes
// example input channel looks like "{{$.inputs.parameters['pipelinechannel--val']}}"
const inputPipelineChannelPattern = `\$.inputs.parameters\['(.+?)'\]`

func isInputParameterChannel(inputChannel string) bool {
	re := regexp.MustCompile(inputPipelineChannelPattern)
	match := re.FindStringSubmatch(inputChannel)
	if len(match) == 2 {
		return true
	} else {
		// if len(match) > 2, then this is still incorrect because
		// inputChannel should contain only one parameter channel input
		return false
	}
}

// extractInputParameterFromChannel takes an inputChannel that adheres to
// inputPipelineChannelPattern and extracts the channel parameter name.
// For example given an input channel of the form "{{$.inputs.parameters['pipelinechannel--val']}}"
// the channel parameter name "pipelinechannel--val" is returned.
func extractInputParameterFromChannel(inputChannel string) (string, error) {
	re := regexp.MustCompile(inputPipelineChannelPattern)
	match := re.FindStringSubmatch(inputChannel)
	if len(match) > 1 {
		extractedValue := match[1]
		return extractedValue, nil
	} else {
		return "", fmt.Errorf("failed to extract input parameter from channel: %s", inputChannel)
	}
}

// inputParamConstant convert and return value as a RuntimeValue
func inputParamConstant(value string) *pipelinespec.TaskInputsSpec_InputParameterSpec {
	return &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
			RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue(value),
				},
			},
		},
	}
}

// inputParamComponent convert and return value as a ComponentInputParameter
func inputParamComponent(value string) *pipelinespec.TaskInputsSpec_InputParameterSpec {
	return &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
			ComponentInputParameter: value,
		},
	}
}

// inputParamTaskOutput convert and return producerTask & outputParamKey
// as a TaskOutputParameter.
func inputParamTaskOutput(producerTask, outputParamKey string) *pipelinespec.TaskInputsSpec_InputParameterSpec {
	return &pipelinespec.TaskInputsSpec_InputParameterSpec{
		Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter{
			TaskOutputParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec{
				ProducerTask:       producerTask,
				OutputParameterKey: outputParamKey,
			},
		},
	}
}

// Get iteration items from a structpb.Value.
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

// validateRootDAG contains validation for root DAG driver options, without MLMD dependencies.
func validateRootDAG(opts common.Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid root DAG driver args: %w", err)
		}
	}()
	if opts.PipelineName == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if opts.Run.GetRunId() == "" {
		return fmt.Errorf("KFP run ID is required")
	}
	if opts.Component == nil {
		return fmt.Errorf("component spec is required")
	}
	if opts.RuntimeConfig == nil {
		return fmt.Errorf("runtime config is required")
	}
	if opts.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if opts.Task.GetTaskInfo().GetName() != "" {
		return fmt.Errorf("task spec is unnecessary")
	}
	if opts.ParentTask != nil && opts.ParentTask.GetTaskId() == "" {
		return fmt.Errorf("Parent task is required")
	}
	if opts.Container != nil {
		return fmt.Errorf("container spec is unnecessary")
	}
	if opts.IterationIndex >= 0 {
		return fmt.Errorf("iteration index is unnecessary")
	}
	return nil
}

// validateDAG validates non-root DAG options without MLMD.
func validateDAG(opts common.Options) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("invalid DAG driver args: %w", err)
		}
	}()
	if opts.Container != nil {
		return fmt.Errorf("container spec is unnecessary")
	}
	return validateNonRoot(opts)
}

func validateNonRoot(opts common.Options) error {
	if opts.PipelineName == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if opts.Run.GetRunId() == "" {
		return fmt.Errorf("KFP run ID is required")
	}
	if opts.Component == nil {
		return fmt.Errorf("component spec is required")
	}
	if opts.Task.GetTaskInfo().GetName() == "" {
		return fmt.Errorf("task spec is required")
	}
	if opts.RuntimeConfig != nil {
		return fmt.Errorf("runtime config is unnecessary")
	}
	if opts.ParentTask != nil && opts.ParentTask.GetTaskId() == "" {
		return fmt.Errorf("Parent task is required")
	}
	return nil
}

func parsePipelineChannelName(name string) (key string, producerTaskName string, producerKey string) {
	const prefix = "pipelinechannel--"
	if !strings.HasPrefix(name, prefix) {
		return name, "", ""
	}

	// Remove prefix
	nameWithoutPrefix := strings.TrimPrefix(name, prefix)

	// Split remaining string by last dash to separate producer task name and key
	parts := strings.Split(nameWithoutPrefix, "-")
	if len(parts) < 2 {
		return nameWithoutPrefix, "", ""
	}

	key = parts[len(parts)-1]
	producerTaskName = strings.Join(parts[:len(parts)-1], "-")
	producerKey = key

	return key, producerTaskName, producerKey
}

// handleTaskParametersCreation creates a new PipelineTaskDetail_InputOutputs_Parameter
// for each parameter in the executor input.
func handleTaskParametersCreation(
	executorInput *pipelinespec.ExecutorInput,
	task *apiV2beta1.PipelineTaskDetail,
) (*apiV2beta1.PipelineTaskDetail, error) {

	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	if task.Inputs == nil {
		task.Inputs = &apiV2beta1.PipelineTaskDetail_InputOutputs{
			Parameters: []*apiV2beta1.PipelineTaskDetail_InputOutputs_Parameter{},
		}
	} else if task.Inputs.Parameters == nil {
		task.Inputs.Parameters = []*apiV2beta1.PipelineTaskDetail_InputOutputs_Parameter{}
	}

	for parameterName, parameter := range executorInput.Inputs.ParameterValues {
		// We expect that a parameter is either a pipelinechannel parameter or a regular parameter.
		// in the latter case we expect that the parameter name is the key.
		key, producerTaskName, producerKey := parsePipelineChannelName(parameterName)
		if (producerTaskName == "" && producerKey != "") || (producerTaskName != "" && producerKey == "") {
			return nil, fmt.Errorf("either both producerTaskName and producerKey must be specified, or both must be empty")
		}
		parameterNew := &apiV2beta1.PipelineTaskDetail_InputOutputs_Parameter{
			Value: parameter,
		}
		if key != "" {
			if producerTaskName != "" {
				return nil, fmt.Errorf("producer and key cannot be specified at the same time")
			}
			parameterNew.Source = &apiV2beta1.PipelineTaskDetail_InputOutputs_Parameter_ParameterName{
				ParameterName: key,
			}
		} else {
			parameterNew.Source = &apiV2beta1.PipelineTaskDetail_InputOutputs_Parameter_Producer{
				Producer: &apiV2beta1.PipelineTaskDetail_InputOutputs_IOProducer{
					TaskName: producerTaskName,
					Key:      producerKey,
				},
			}
		}
		task.Inputs.Parameters = append(task.Inputs.Parameters, parameterNew)
	}
	return task, nil
}
func handleTaskArtifactsCreation(
	ctx context.Context,
	executorInput *pipelinespec.ExecutorInput,
	opts common.Options,
	task *apiV2beta1.PipelineTaskDetail,
	driverAPI common.DriverAPI,
) error {
	var artifactTasks []*apiV2beta1.ArtifactTask
	for parameterName, artifactList := range executorInput.Inputs.Artifacts {
		for _, artifact := range artifactList.Artifacts {
			if artifact.ArtifactId == "" {
				return fmt.Errorf("artifact id is required")
			}
			key, producerTaskName, producerKey := parsePipelineChannelName(parameterName)
			at := &apiV2beta1.ArtifactTask{
				ArtifactId:       artifact.ArtifactId,
				RunId:            opts.Run.GetRunId(),
				TaskId:           task.TaskId,
				Type:             apiV2beta1.ArtifactTaskType_INPUT,
				ArtifactKey:      key,
				ProducerTaskName: producerTaskName,
				ProducerKey:      producerKey,
			}
			artifactTasks = append(artifactTasks, at)
		}
	}
	if len(artifactTasks) > 0 {
		request := apiV2beta1.CreateArtifactTasksBulkRequest{ArtifactTasks: artifactTasks}
		_, err := driverAPI.CreateArtifactTasks(ctx, &request)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseIONameOrPipelineChannel(name string, producer *apiV2beta1.PipelineTaskDetail_InputOutputs_IOProducer) (string, error) {
	var result string
	if producer != nil {
		if producer.GetTaskName() == "" || producer.Key == "" {
			return "", fmt.Errorf("producer task name or key is empty")
		}
		result = fmt.Sprintf("pipelinechannel--%s-%s", producer.GetTaskName(), producer.Key)
	} else if name != "" {
		result = name
	} else {
		return "", fmt.Errorf("producer task name or key is empty")
	}
	return result, nil
}
