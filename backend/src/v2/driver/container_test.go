package driver

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func setupOptions(
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

	// Run Container on the First Task
	taskSpec := pipelineSpec.Root.GetDag().Tasks["process-inputs"]
	opts := setupOptions(t, testSetup, run, parentTask, taskSpec, pipelineSpec, nil)
	execution, err := Container(context.Background(), opts, testSetup.DriverAPI)
	require.NoError(t, err)
	require.NotNil(t, execution)
	require.Nil(t, execution.ExecutorInput.Outputs)
	taskResp, err := testSetup.DriverAPI.ListTasks(
		context.Background(),
		&apiv2beta1.ListTasksRequest{
			ParentFilter: &apiv2beta1.ListTasksRequest_ParentId{
				ParentId: parentTask.TaskId,
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, taskResp)
	require.Equal(t, 1, len(taskResp.Tasks))
	processInputsTask := taskResp.Tasks[0]
	require.Equal(t, execution.TaskID, processInputsTask.TaskId)
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["name"].GetStringValue(), "some_name")
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["number"].GetNumberValue(), 1.0)
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["threshold"].GetNumberValue(), 0.1)
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["active"].GetBoolValue(), false)
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["a_runtime_string"].GetStringValue(), "foo")
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["a_runtime_number"].GetNumberValue(), 10.0)
	require.Equal(t, execution.ExecutorInput.Inputs.ParameterValues["a_runtime_bool"].GetBoolValue(), true)

	// Mock a Launcher run by updating the task with output data
	processInputsTask.Outputs = &apiv2beta1.PipelineTaskDetail_InputOutputs{
		Artifacts: []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
			{
				Source: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact_Producer{
					Producer: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOProducer{
						TaskName: "process-inputs",
						Key:      "output_text",
					},
				},
			},
		},
	}
	processInputsTask.Status = apiv2beta1.PipelineTaskDetail_SUCCEEDED
	task, err := testSetup.DriverAPI.UpdateTask(context.Background(), &apiv2beta1.UpdateTaskRequest{
		TaskId: processInputsTask.TaskId,
		Task:   processInputsTask,
	})
	require.NoError(t, err)
	require.NotNil(t, task)

	// Run the Downstream Task that will use the output artifact
	taskSpec = pipelineSpec.Root.GetDag().Tasks["analyze-inputs"]
	opts = setupOptions(t, testSetup, run, parentTask, taskSpec, pipelineSpec, nil)
	execution, err = Container(context.Background(), opts, testSetup.DriverAPI)
	require.NoError(t, err)
	require.NotNil(t, execution)
	require.Nil(t, execution.ExecutorInput.Outputs)
	taskResp, err = testSetup.DriverAPI.ListTasks(
		context.Background(),
		&apiv2beta1.ListTasksRequest{
			ParentFilter: &apiv2beta1.ListTasksRequest_ParentId{
				ParentId: parentTask.TaskId,
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, taskResp)
	require.Equal(t, 2, len(taskResp.Tasks))

}

// TODO(HumairAK):
// Caching tests
// Optional fields
