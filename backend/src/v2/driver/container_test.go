package driver

import (
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func fetchParameter(key string, params []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter) *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
	for _, p := range params {
		if key == p.ParameterKey {
			return p
		}
	}
	panic("parameter not found")
}

// This test creates a DAG with a single task that uses a component with inputs
// and runtime constants. The test verifies that the inputs are correctly passed
// to the Runtime Task.
func TestContainerComponentInputsAndRuntimeConstants(t *testing.T) {
	// Create a root DAG execution using basic inputs
	runtimeInputs := &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues: map[string]*structpb.Value{
			"name_in":      structpb.NewStringValue("some_name"),
			"number_in":    structpb.NewNumberValue(1.0),
			"threshold_in": structpb.NewNumberValue(0.1),
			"active_in":    structpb.NewBoolValue(false),
		},
	}

	currentRun := SetupCurrentRun(t, runtimeInputs, "test_data/componentInput_level_1_test.py.yaml")

	// Run Container on the First Task
	processInputsExecution, processInputsTask := currentRun.RunContainer("process-inputs", currentRun.RootTask, nil)
	require.Nil(t, processInputsExecution.ExecutorInput.Outputs)

	// Fetch the task created by the Container() call
	params := processInputsTask.Inputs.GetParameters()
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, fetchParameter("name", params).GetType())
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, fetchParameter("number", params).GetType())
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, fetchParameter("active", params).GetType())
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, fetchParameter("threshold", params).GetType())
	require.Equal(t, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, fetchParameter("a_runtime_string", params).GetType())
	require.Equal(t, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, fetchParameter("a_runtime_number", params).GetType())
	require.Equal(t, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, fetchParameter("a_runtime_bool", params).GetType())

	require.Equal(t, processInputsExecution.TaskID, processInputsTask.TaskId)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["name"].GetStringValue(), "some_name")
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["number"].GetNumberValue(), 1.0)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["threshold"].GetNumberValue(), 0.1)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["active"].GetBoolValue(), false)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["a_runtime_string"].GetStringValue(), "foo")
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["a_runtime_number"].GetNumberValue(), 10.0)
	require.Equal(t, processInputsExecution.ExecutorInput.Inputs.ParameterValues["a_runtime_bool"].GetBoolValue(), true)

	// Mock a Launcher run by updating the task with output data
	currentRun.MockLauncherArtifactCreate(
		processInputsTask.TaskId,
		"output_text",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		"process-inputs",
		nil,
	)

	analyzeInputsExecution, _ := currentRun.RunContainer("analyze-inputs", currentRun.RootTask, nil)
	require.Nil(t, analyzeInputsExecution.ExecutorInput.Outputs)
	require.Nil(t, analyzeInputsExecution.ExecutorInput.Outputs)
	require.Equal(t, 1, len(analyzeInputsExecution.ExecutorInput.Inputs.Artifacts["input_text"].Artifacts))

	// Verify Executor Input has the correct artifact
	artifact := analyzeInputsExecution.ExecutorInput.Inputs.Artifacts["input_text"].Artifacts[0]
	require.NotNil(t, artifact.Metadata)
	require.NotNil(t, artifact.Metadata.GetFields()["display_name"])
	require.Equal(t, "output_text", artifact.Metadata.GetFields()["display_name"].GetStringValue())
	require.Equal(t, "s3://some.location/output_text", artifact.Uri)
	require.Equal(t, apiv2beta1.Artifact_Dataset.String(), artifact.Type.GetSchemaTitle())
	require.Equal(t, "output_text", artifact.Name)
}

// TODO: Add tests for optional fields
func TestOptionalFields(t *testing.T) {}
