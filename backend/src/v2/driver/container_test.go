package driver

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func setupContainerOptions(
	t *testing.T,
	testSetup *TestSetup,
	run *apiv2beta1.Run,
	parentTask *apiv2beta1.PipelineTaskDetail,
	taskSpec *pipelinespec.PipelineTaskSpec,
	pipelineSpec *pipelinespec.PipelineSpec,
	KubernetesExecutorConfig *kubernetesplatform.KubernetesExecutorConfig,
) common.Options {
	componentSpec := pipelineSpec.Components[taskSpec.ComponentRef.Name]

	ds := pipelineSpec.GetDeploymentSpec()
	platformDeploymentSpec := &pipelinespec.PlatformDeploymentConfig{}

	b, err := protojson.Marshal(ds)
	require.NoError(t, err)
	err = protojson.Unmarshal(b, platformDeploymentSpec)
	require.NoError(t, err)
	assert.NotNil(t, platformDeploymentSpec)

	cs := platformDeploymentSpec.Executors[componentSpec.GetExecutorLabel()]
	containerExecutorSpec := &pipelinespec.PipelineDeploymentConfig_ExecutorSpec{}
	b, err = protojson.Marshal(cs)
	require.NoError(t, err)
	err = protojson.Unmarshal(b, containerExecutorSpec)
	require.NoError(t, err)
	assert.NotNil(t, containerExecutorSpec)

	return common.Options{
		PipelineName:             testPipelineName,
		Run:                      run,
		Component:                componentSpec,
		ParentTask:               parentTask,
		DriverAPI:                testSetup.DriverAPI,
		IterationIndex:           -1,
		RuntimeConfig:            nil,
		Namespace:                testNamespace,
		Task:                     taskSpec,
		Container:                containerExecutorSpec.GetContainer(),
		KubernetesExecutorConfig: KubernetesExecutorConfig,
		PipelineLogLevel:         "1",
		PublishLogs:              "false",
		CacheDisabled:            false,
		DriverType:               "CONTAINER",
		TaskName:                 taskSpec.TaskInfo.GetName(),
		PodName:                  "system-container-impl",
		PodUID:                   "some-uid",
	}
}

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
	// Setup test environment
	testSetup := NewTestSetup(t)

	// Create a test run
	run := testSetup.CreateTestRun(t, "test-pipeline")
	require.NotNil(t, run)

	// Load pipeline spec
	pipelineSpec, err := LoadPipelineSpecFromYAML("test_data/componentInput_level_1_test.py.yaml")
	require.NoError(t, err)
	require.NotNil(t, pipelineSpec)

	// Create a root DAG execution using basic inputs
	runtimeInputs := &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues: map[string]*structpb.Value{
			"name_in":      structpb.NewStringValue("some_name"),
			"number_in":    structpb.NewNumberValue(1.0),
			"threshold_in": structpb.NewNumberValue(0.1),
			"active_in":    structpb.NewBoolValue(false),
		},
	}
	rootDagExecution, err := setupBasicRootDag(testSetup, run, pipelineSpec, runtimeInputs)
	require.NoError(t, err)
	require.NotNil(t, rootDagExecution)

	parentTask, err := testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: rootDagExecution.TaskID,
	})
	require.NoError(t, err)

	// Refresh Run so it has the new tasks
	run, err = testSetup.DriverAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.GetRunId()})
	require.NoError(t, err)

	// Run Container on the First Task
	taskSpec := pipelineSpec.Root.GetDag().Tasks["process-inputs"]
	opts := setupContainerOptions(t, testSetup, run, parentTask, taskSpec, pipelineSpec, nil)
	execution, err := Container(context.Background(), opts, testSetup.DriverAPI)
	require.NoError(t, err)
	require.NotNil(t, execution)
	require.Nil(t, execution.ExecutorInput.Outputs)

	// Fetch the task created by the Container() call
	processInputsTask, err := testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: execution.TaskID})
	require.NoError(t, err)
	require.NotNil(t, processInputsTask)
	params := processInputsTask.Inputs.GetParameters()
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, fetchParameter("name", params).GetType())
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, fetchParameter("number", params).GetType())
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, fetchParameter("active", params).GetType())
	require.Equal(t, apiv2beta1.IOType_COMPONENT_INPUT, fetchParameter("threshold", params).GetType())
	require.Equal(t, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, fetchParameter("a_runtime_string", params).GetType())
	require.Equal(t, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, fetchParameter("a_runtime_number", params).GetType())
	require.Equal(t, apiv2beta1.IOType_RUNTIME_VALUE_INPUT, fetchParameter("a_runtime_bool", params).GetType())

	require.Equal(t, execution.TaskID, processInputsTask.TaskId)
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["name"].GetStringValue(), "some_name")
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["number"].GetNumberValue(), 1.0)
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["threshold"].GetNumberValue(), 0.1)
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["active"].GetBoolValue(), false)
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["a_runtime_string"].GetStringValue(), "foo")
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["a_runtime_number"].GetNumberValue(), 10.0)
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["a_runtime_bool"].GetBoolValue(), true)

	// Mock a Launcher run by updating the task with output data
	outputArtifact := &apiv2beta1.Artifact{
		ArtifactId: "some-artifact-id-1",
		Name:       "output_text",
		Type:       apiv2beta1.Artifact_Dataset,
		Uri:        util.StringPointer("s3://some.location/output_text"),
		Namespace:  testNamespace,
		Metadata: map[string]*structpb.Value{
			"display_name": structpb.NewStringValue("output_text"),
		},
	}
	createArtifact, err := testSetup.DriverAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{Artifact: outputArtifact})
	require.NoError(t, err)
	require.NotNil(t, createArtifact)
	at, err := testSetup.DriverAPI.CreateArtifactTask(context.Background(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: &apiv2beta1.ArtifactTask{
			ArtifactId: createArtifact.ArtifactId,
			TaskId:     processInputsTask.TaskId,
			RunId:      run.GetRunId(),
			Key:        "output_text",
			Producer:   &apiv2beta1.IOProducer{TaskName: "process-inputs"},
			Type:       apiv2beta1.IOType_OUTPUT,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, at)

	// Refresh Run so it has the new tasks
	run, err = testSetup.DriverAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.GetRunId()})
	require.NoError(t, err)

	// Run the Downstream Task that will use the output artifact
	taskSpec = pipelineSpec.Root.GetDag().Tasks["analyze-inputs"]
	opts = setupContainerOptions(t, testSetup, run, parentTask, taskSpec, pipelineSpec, nil)
	execution, err = Container(context.Background(), opts, testSetup.DriverAPI)
	require.NoError(t, err)
	require.NotNil(t, execution)
	require.Nil(t, execution.ExecutorInput.Outputs)

	// Fetch the task created by the Container() call
	analyzeTaskResp, err := testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: execution.TaskID})
	require.NoError(t, err)
	require.Equal(t, execution.TaskID, analyzeTaskResp.TaskId)
	require.Equal(t, 1, len(execution.ExecutorInput.Inputs.Artifacts["input_text"].Artifacts))

	// Verify Executor Input has the correct artifact
	artifact := execution.ExecutorInput.Inputs.Artifacts["input_text"].Artifacts[0]
	require.NotNil(t, artifact.Metadata)
	require.NotNil(t, artifact.Metadata.GetFields()["display_name"])
	require.Equal(t, "output_text", artifact.Metadata.GetFields()["display_name"].GetStringValue())
	require.Equal(t, "s3://some.location/output_text", artifact.Uri)
	require.Equal(t, apiv2beta1.Artifact_Dataset.String(), artifact.Type.GetSchemaTitle())
	require.Equal(t, "output_text", artifact.Name)
}

// Try with multiple upstream tasks, but one has the correct input (maybe just update the above)

// TODO(HumairAK):
// Do an output that points to a Dag (for-loop-2)
// Collect inputs
// Caching tests
// Optional fields
