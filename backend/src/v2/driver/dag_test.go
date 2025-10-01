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

func setupDagOptions(
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
		Container:                nil,
		KubernetesExecutorConfig: KubernetesExecutorConfig,
		RunName:                  "",
		RunDisplayName:           "",
		PipelineLogLevel:         "1",
		PublishLogs:              "false",
		CacheDisabled:            false,
		DriverType:               "DAG",
		TaskName:                 taskSpec.TaskInfo.GetName(),
		PodName:                  "system-dag-driver",
		PodUID:                   "some-uid",
	}
}

func TestLoopArtifactPassing(t *testing.T) {
	// Setup test environment
	testSetup := NewTestSetup(t)

	// Create a test run
	run := testSetup.CreateTestRun(t, "test-pipeline")
	require.NotNil(t, run)

	// Load pipeline spec
	pipelineSpec, err := LoadPipelineSpecFromYAML("test_data/loop_collected.py.yaml")
	require.NoError(t, err)
	require.NotNil(t, pipelineSpec)

	// Create a root DAG execution using basic inputs
	rootDagExecution, err := setupBasicRootDag(testSetup, run, pipelineSpec, &pipelinespec.PipelineJob_RuntimeConfig{})
	require.NoError(t, err)
	require.NotNil(t, rootDagExecution)

	parentTask, err := testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: rootDagExecution.TaskID,
	})
	require.NoError(t, err)

	// Refresh Run so it has the new tasks
	run, err = testSetup.DriverAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.GetRunId()})
	require.NoError(t, err)

	// Run Dag on the First Task
	secondaryPipelineTaskSpec := pipelineSpec.Root.GetDag().Tasks["secondary-pipeline"]
	opts := setupDagOptions(t, testSetup, run, parentTask, secondaryPipelineTaskSpec, pipelineSpec, nil)
	secondaryPipelineExecution, err := DAG(context.Background(), opts, testSetup.DriverAPI)
	require.NoError(t, err)
	require.NotNil(t, secondaryPipelineExecution)
	require.Nil(t, secondaryPipelineExecution.ExecutorInput.Outputs)

	// Fetch the task created by the Dag() call
	secondaryPipelineTask, err := testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: secondaryPipelineExecution.TaskID})
	require.NoError(t, err)
	require.NoError(t, err)
	require.NotNil(t, secondaryPipelineTask)
	require.Equal(t, secondaryPipelineExecution.TaskID, secondaryPipelineTask.TaskId)
	require.Equal(t, "secondary-pipeline", secondaryPipelineTask.GetName())
	require.Equal(t, apiv2beta1.PipelineTaskDetail_RUNNING, secondaryPipelineTask.Status)

	// Refresh Run
	run, err = testSetup.DriverAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.GetRunId()})
	require.NoError(t, err)
	// Refresh Parent Task - The parent task should be the secondary pipeline task for "create-dataset"
	parentTask, err = testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: secondaryPipelineExecution.TaskID})
	require.NoError(t, err)

	// In this section we'll run the subtasks in the secondary pipeline, one of which is a loop of 3 iterations

	// Run the Downstream Task that will use the output artifact
	createDatasetTaskSpec := pipelineSpec.Components["comp-secondary-pipeline"].GetDag().Tasks["create-dataset"]
	opts = setupContainerOptions(t, testSetup, run, parentTask, createDatasetTaskSpec, pipelineSpec, nil)
	createDataSetExecution, err := Container(context.Background(), opts, testSetup.DriverAPI)
	require.NoError(t, err)
	require.NotNil(t, createDataSetExecution)
	require.Nil(t, createDataSetExecution.ExecutorInput.Outputs)

	// Mock a Launcher run by updating the task with output data
	outputArtifact := &apiv2beta1.Artifact{
		ArtifactId: "some-artifact-id-1",
		Name:       "output_dataset",
		Type:       apiv2beta1.Artifact_Dataset,
		Uri:        util.StringPointer("s3://some.location/output_dataset"),
		Namespace:  testNamespace,
		Metadata: map[string]*structpb.Value{
			"display_name": structpb.NewStringValue("output_dataset"),
		},
	}
	createArtifact, err := testSetup.DriverAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{Artifact: outputArtifact})
	require.NoError(t, err)
	require.NotNil(t, createArtifact)
	at, err := testSetup.DriverAPI.CreateArtifactTask(context.Background(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: &apiv2beta1.ArtifactTask{
			ArtifactId:       createArtifact.ArtifactId,
			TaskId:           createDataSetExecution.TaskID,
			RunId:            run.GetRunId(),
			ProducerKey:      "output_dataset",
			ProducerTaskName: "create-dataset",
			Type:             apiv2beta1.IOType_OUTPUT,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, at)

	// Refresh Run so it has the new tasks
	run, err = testSetup.DriverAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.GetRunId()})
	require.NoError(t, err)

	// Run the Loop Task - note that parentTask for for-loop-2 remains as secondary-pipeline
	loopTaskSpec := pipelineSpec.Components["comp-secondary-pipeline"].GetDag().Tasks["for-loop-2"]
	opts = setupDagOptions(t, testSetup, run, parentTask, loopTaskSpec, pipelineSpec, nil)
	loopExecution, err := DAG(context.Background(), opts, testSetup.DriverAPI)
	require.NoError(t, err)
	require.NotNil(t, loopExecution)
	require.Nil(t, loopExecution.ExecutorInput.Outputs)

	// ---------------------
	// Run the first iteration
	// Refresh Run so it has the new tasks
	run, err = testSetup.DriverAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.GetRunId()})
	require.NoError(t, err)

	// Refresh Parent Task - The parent task should be "for-loop-2" for "process-dataset"
	parentTask, err = testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: loopExecution.TaskID})
	require.NoError(t, err)

	// Run the "process-dataset" Container Task with iteration index
	processTaskSpec := pipelineSpec.Components["comp-for-loop-2"].GetDag().Tasks["process-dataset"]
	opts = setupContainerOptions(t, testSetup, run, parentTask, processTaskSpec, pipelineSpec, nil)
	opts.IterationIndex = 0
	processExecution, err := Container(context.Background(), opts, testSetup.DriverAPI)
	require.NoError(t, err)
	require.NotNil(t, processExecution)
	require.Nil(t, processExecution.ExecutorInput.Outputs)
	require.NotNil(t, processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"])
	require.Equal(t, 1, len(processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"].GetArtifacts()))
	require.Equal(t,
		processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"].GetArtifacts()[0].ArtifactId,
		"some-artifact-id-1",
	)
	require.NotNil(t, processExecution.ExecutorInput.Inputs.ParameterValues["model_id_in"])
	require.Equal(t,
		processExecution.ExecutorInput.Inputs.ParameterValues["model_id_in"].GetStringValue(),
		"1",
	)

	// Mock the Launcher run
	outputArtifact = &apiv2beta1.Artifact{
		ArtifactId: "some-artifact-id-2",
		Name:       "output_artifact",
		Type:       apiv2beta1.Artifact_Artifact,
		Uri:        util.StringPointer("s3://some.location/output_artifact"),
		Namespace:  testNamespace,
		Metadata: map[string]*structpb.Value{
			"display_name": structpb.NewStringValue("output_artifact"),
		},
	}
	createArtifact, err = testSetup.DriverAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{Artifact: outputArtifact})
	require.NoError(t, err)
	require.NotNil(t, createArtifact)
	at, err = testSetup.DriverAPI.CreateArtifactTask(context.Background(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: &apiv2beta1.ArtifactTask{
			ArtifactId:       createArtifact.ArtifactId,
			TaskId:           processExecution.TaskID,
			RunId:            run.GetRunId(),
			ProducerKey:      "output_artifact",
			ProducerTaskName: "process-dataset",
			Type:             apiv2beta1.IOType_OUTPUT,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, at)

	// Refresh Run so it has the new tasks
	run, err = testSetup.DriverAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.GetRunId()})
	require.NoError(t, err)

	// Run next iteration component
	analyzeTaskSpec := pipelineSpec.Components["comp-for-loop-2"].GetDag().Tasks["analyze-artifact"]
	opts = setupContainerOptions(t, testSetup, run, parentTask, analyzeTaskSpec, pipelineSpec, nil)
	opts.IterationIndex = 0
	analyzeExecution, err := Container(context.Background(), opts, testSetup.DriverAPI)
	require.NoError(t, err)
	require.NotNil(t, analyzeExecution)
	require.Nil(t, analyzeExecution.ExecutorInput.Outputs)
	require.NotNil(t, analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"])
	require.Equal(t, 1, len(analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"].GetArtifacts()))
	require.Equal(t, analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"].GetArtifacts()[0].ArtifactId, "some-artifact-id-2")

	outputArtifact = &apiv2beta1.Artifact{
		ArtifactId: "some-artifact-id-3",
		Name:       "analyze_output_artifact",
		Type:       apiv2beta1.Artifact_Artifact,
		Uri:        util.StringPointer("s3://some.location/analyze_output_artifact"),
		Namespace:  testNamespace,
		Metadata: map[string]*structpb.Value{
			"display_name": structpb.NewStringValue("analyze_output_artifact"),
		},
	}
	createArtifact, err = testSetup.DriverAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{Artifact: outputArtifact})
	require.NoError(t, err)
	require.NotNil(t, createArtifact)
	at, err = testSetup.DriverAPI.CreateArtifactTask(context.Background(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: &apiv2beta1.ArtifactTask{
			ArtifactId:       createArtifact.ArtifactId,
			TaskId:           analyzeExecution.TaskID,
			RunId:            run.GetRunId(),
			ProducerKey:      "analyze_output_artifact",
			ProducerTaskName: "analyze-artifact",
			Type:             apiv2beta1.IOType_OUTPUT,
		},
	})
	// ---------------------

	// Run the second iteration
	run, err = testSetup.DriverAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.GetRunId()})
	require.NoError(t, err)

	// Run the third iteration
	run, err = testSetup.DriverAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.GetRunId()})
	require.NoError(t, err)

	// TODO:
	//analyzeArtifactListTaskSpec := pipelineSpec.Components["comp-secondary-pipeline"].GetDag().Tasks["analyze-artifact-list"]
	//opts = setupContainerOptions(t, testSetup, run, parentTask, analyzeArtifactListTaskSpec, pipelineSpec, nil)
	//analyzeArtifactListExecution, err := Container(context.Background(), opts, testSetup.DriverAPI)
	//require.NoError(t, err)
	//require.NotNil(t, analyzeArtifactListExecution)
	//require.Nil(t, analyzeArtifactListExecution.ExecutorInput.Outputs)
	//// Assert that "artifact_list_input"'s executor input contains the artifact "artifact_list_input" and it is a list of artifacts
}
