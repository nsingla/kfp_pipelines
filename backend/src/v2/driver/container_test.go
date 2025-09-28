package driver

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func setupOptions(
	t *testing.T,
	testSetup *TestSetup,
	run *apiv2beta1.Run,
	parentTask *apiv2beta1.PipelineTaskDetail,
	taskSpec *pipelinespec.PipelineTaskSpec,
	pipelineSpec *pipelinespec.PipelineSpec,
	KubernetesExecutorConfig *kubernetesplatform.KubernetesExecutorConfig,
) Options {
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

	return Options{
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

func TestContainerComponentInputs(t *testing.T) {
	// Setup test environment
	testSetup := NewTestSetup(t)

	// Create a test run
	run := testSetup.CreateTestRun(t, "test-pipeline")
	assert.NotNil(t, run)

	pipelineSpec, err := LoadPipelineSpecFromYAML("test_data/taskOutput_level_1_test.py.yaml")
	require.NoError(t, err)
	require.NotNil(t, pipelineSpec)

	rootDagExecution, err := setupBasicRootDag(testSetup, run, pipelineSpec, basicRuntimeConfig())
	require.NoError(t, err)
	require.NotNil(t, rootDagExecution)

	parentTask, err := testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: rootDagExecution.TaskID,
	})
	require.NoError(t, err)

	taskSpec := pipelineSpec.Root.GetDag().Tasks["process-dataset"]
	opts := setupOptions(t, testSetup, run, parentTask, taskSpec, pipelineSpec, nil)

	execution, err := Container(context.Background(), opts, testSetup.DriverAPI)
	require.NoError(t, err)
	require.NotNil(t, execution)
}
