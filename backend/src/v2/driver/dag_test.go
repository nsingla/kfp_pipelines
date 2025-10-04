package driver

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
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

type CurrentRun struct {
	Run *apiv2beta1.Run
	ScopePath
	T            *testing.T
	TestSetup    *TestSetup
	PipelineSpec *pipelinespec.PipelineSpec
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

	rootTask, err := testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: rootDagExecution.TaskID,
	})
	require.NoError(t, err)
	parentTask := rootTask

	currentRun := &CurrentRun{
		Run:          run,
		ScopePath:    NewScopePath(pipelineSpec),
		T:            t,
		TestSetup:    testSetup,
		PipelineSpec: pipelineSpec,
	}
	err = currentRun.ScopePath.Push("root")
	require.NoError(t, err)

	// Run Dag on the First Task
	secondaryPipelineExecution, secondaryPipelineTask := currentRun.RunDag("secondary-pipeline", parentTask)
	require.Nil(t, secondaryPipelineExecution.ExecutorInput.Outputs)
	require.Equal(t, apiv2beta1.PipelineTaskDetail_RUNNING, secondaryPipelineTask.Status)

	// Refresh Parent Task - The parent task should be the secondary pipeline task for "create-dataset"
	parentTask = secondaryPipelineTask

	// Now we'll run the subtasks in the secondary pipeline, one of which is a loop of 3 iterations

	// Run the Downstream Task that will use the output artifact
	createDataSetExecution, _ := currentRun.RunContainer("create-dataset", parentTask, nil)
	require.Nil(t, createDataSetExecution.ExecutorInput.Outputs)

	// Mock a Launcher run by updating the task with output data
	createDataSetOutputArtifactID := currentRun.MockLauncherArtifactCreate(
		createDataSetExecution.TaskID,
		"output_dataset",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		"create-dataset",
		nil,
	)

	// Run the Loop Task - note that parentTask for for-loop-2 remains as secondary-pipeline
	loopExecution, loopTask := currentRun.RunDag("for-loop-2", parentTask)
	require.Nil(t, secondaryPipelineExecution.ExecutorInput.Outputs)
	require.NotZero(t, len(loopTask.Inputs.Parameters))
	// Expect loop task to have resolved its input parameter
	require.Equal(t, "pipelinechannel--loop-item-param-1", loopTask.Inputs.Parameters[0].ParameterKey)
	// Expect the artifact output of create-dataset as input to for-loop-2
	require.Equal(t, len(loopTask.Inputs.Artifacts), 1)

	// The parent task should be "for-loop-2" for the iterations at first depth
	parentTask = loopTask

	// Perform the iteration calls, mock any launcher calls
	for index, paramID := range []string{"1", "2", "3"} {

		// Run the "process-dataset" Container Task with iteration index
		processExecution, _ := currentRun.RunContainer("process-dataset", parentTask, util.Int64Pointer(int64(index)))
		require.Nil(t, processExecution.ExecutorInput.Outputs)
		require.NotNil(t, processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"])
		require.Equal(t, 1, len(processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"].GetArtifacts()))
		require.Equal(t, processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"].GetArtifacts()[0].ArtifactId, createDataSetOutputArtifactID)
		require.NotNil(t, processExecution.ExecutorInput.Inputs.ParameterValues["model_id_in"])
		require.Equal(t, processExecution.ExecutorInput.Inputs.ParameterValues["model_id_in"].GetStringValue(), paramID)

		// Mock the Launcher run
		processDataSetArtifactID := currentRun.MockLauncherArtifactCreate(
			processExecution.TaskID,
			"output_artifact",
			apiv2beta1.Artifact_Artifact,
			apiv2beta1.IOType_OUTPUT,
			"process-dataset",
			util.Int64Pointer(int64(index)),
		)

		// Mock: Also expect Launcher->API Server to upload the output artifact to the for-loop-2 task's outputs (by first checking if this artifact is an output artifact)
		//   comp-for-loop-2:
		//    dag:
		//      outputs:
		//        artifacts:
		//          pipelinechannel--process-dataset-output_artifact:
		//            artifactSelectors:
		//            - outputArtifactKey: output_artifact
		//              producerSubtask: process-dataset
		currentRun.MockLauncherArtifactTaskCreate(
			"process-dataset",
			loopExecution.TaskID,
			"pipelinechannel--process-dataset-output_artifact",
			processDataSetArtifactID,
			util.Int64Pointer(int64(index)),
			apiv2beta1.IOType_ITERATOR_OUTPUT,
		)
		loopTask, err = testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: loopExecution.TaskID})
		require.NoError(t, err)
		require.NotNil(t, loopTask.Outputs)
		require.Equal(t, len(loopTask.Outputs.Artifacts), index+1)

		// Mock: Launcher->API Server should also traverse the dag up, to log any output artifacts that are being sourced from the current loop task
		// In this case, secondary-pipeline requires dsl.Collected() sourced from "process-dataset" outputs that are output from for-loop-2
		//   comp-secondary-pipeline:
		//    dag:
		//      outputs:
		//        artifacts:
		//          Output:
		//            artifactSelectors:
		//            - outputArtifactKey: pipelinechannel--process-dataset-output_artifact
		//              producerSubtask: for-loop-2
		currentRun.MockLauncherArtifactTaskCreate(
			"process-dataset",
			secondaryPipelineExecution.TaskID,
			"Output",
			processDataSetArtifactID,
			util.Int64Pointer(int64(index)),
			apiv2beta1.IOType_ITERATOR_OUTPUT,
		)
		secondaryPipelineTask, err = testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: secondaryPipelineExecution.TaskID})
		require.NoError(t, err)
		require.NotNil(t, secondaryPipelineTask.Outputs)
		require.Equal(t, len(secondaryPipelineTask.Outputs.Artifacts), index+1)

		// Run next iteration component
		analyzeExecution, _ := currentRun.RunContainer("analyze-artifact", parentTask, util.Int64Pointer(int64(index)))
		require.Nil(t, createDataSetExecution.ExecutorInput.Outputs)
		require.Nil(t, analyzeExecution.ExecutorInput.Outputs)
		require.NotNil(t, analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"])
		require.Equal(t, 1, len(analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"].GetArtifacts()))
		require.Equal(t, analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"].GetArtifacts()[0].ArtifactId, processDataSetArtifactID)

		// Mock the Launcher run
		_ = currentRun.MockLauncherArtifactCreate(
			processExecution.TaskID,
			"analyze_output_artifact",
			apiv2beta1.Artifact_Artifact,
			apiv2beta1.IOType_OUTPUT,
			"analyze-artifact",
			util.Int64Pointer(int64(index)),
		)
	}

	tasks, err := testSetup.DriverAPI.ListTasks(context.Background(), &apiv2beta1.ListTasksRequest{
		ParentFilter: &apiv2beta1.ListTasksRequest_ParentId{ParentId: loopExecution.TaskID},
	})
	require.NoError(t, err)
	require.NotNil(t, tasks)
	// Expect 3 tasks for analyze-artifact + 3 tasks for process-dataset
	require.Equal(t, 6, len(tasks.Tasks))

	// Expect the 3 artifacts from process-task to have been collected by the for-loop-2 task
	forLoopTask, err := testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: loopExecution.TaskID})
	require.NoError(t, err)
	require.Equal(t, 3, len(forLoopTask.Outputs.Artifacts))

	// Run "analyze_artifact_list" in "secondary_pipeline"
	// Refresh Run so it has the new tasks
	run, err = testSetup.DriverAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.GetRunId()})
	require.NoError(t, err)

	// Move up a parent
	parentTask = secondaryPipelineTask
	_, ok := currentRun.ScopePath.Pop()
	require.True(t, ok)

	analyzeArtifactListExecution, _ := currentRun.RunContainer("analyze-artifact-list", parentTask, nil)
	require.Nil(t, analyzeArtifactListExecution.ExecutorInput.Outputs)
	require.NotNil(t, analyzeArtifactListExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"])
	require.Equal(t, 3, len(analyzeArtifactListExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"].GetArtifacts()))

	// Primary Pipeline tests

	// Expect the 3 artifacts from process-task to have been collected by the secondary-pipeline task
	secondaryPipelineTask, err = testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: secondaryPipelineExecution.TaskID})
	require.NoError(t, err)
	require.Equal(t, 3, len(secondaryPipelineTask.Outputs.Artifacts))

	// Move up a parent
	parentTask = rootTask
	_, ok = currentRun.ScopePath.Pop()
	require.True(t, ok)

	// Not to be confused with the "analyze-artifact-list" task in secondary pipeline,
	// this is the "analyze-artifact-list" task in the primary pipeline
	analyzeArtifactListOuterExecution, _ := currentRun.RunContainer("analyze-artifact-list", parentTask, nil)
	require.Nil(t, analyzeArtifactListExecution.ExecutorInput.Outputs)
	require.Nil(t, analyzeArtifactListOuterExecution.ExecutorInput.Outputs)
	require.NotNil(t, analyzeArtifactListOuterExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"])
	require.Equal(t, 3, len(analyzeArtifactListOuterExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"].GetArtifacts()))

	// Refresh Run so it has the new tasks
	run, err = testSetup.DriverAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.GetRunId()})
	require.NoError(t, err)
	// primary_pipeline()		 x 1  (root)
	// secondary_pipeline()      x 1  (dag)
	//   create_dataset()        x 1  (runtime)
	//   for_loop_1()            x 1  (loop)
	//     process_dataset()     x 3  (runtime)
	//	   analyze_artifact()    x 3  (runtime)
	//   analyze_artifact_list() x 1  (runtime)
	// analyze_artifact_list()   x 1  (runtime)
	require.Equal(t, 12, len(run.Tasks))
}

func TestParameterInputIterator(t *testing.T) {

}

func TestArtifactIterator(t *testing.T) {

}

func TestFinalStatus(t *testing.T) {

}

// TODO Can't do this unless we can mock the filter API call on fingerprint and status
// See getFingerPrintsANDID
func TestWithCaching(t *testing.T) {

}

func TestParameterTaskOutput(t *testing.T) {
	// Setup test environment
	testSetup := NewTestSetup(t)

	// Create a test run
	run := testSetup.CreateTestRun(t, "test-pipeline")
	require.NotNil(t, run)

	// Load pipeline spec
	pipelineSpec, err := LoadPipelineSpecFromYAML("test_data/taskOutputParameter_test.py.yaml")
	require.NoError(t, err)
	require.NotNil(t, pipelineSpec)

	// Create a root DAG execution using basic inputs
	rootDagExecution, err := setupBasicRootDag(testSetup, run, pipelineSpec, &pipelinespec.PipelineJob_RuntimeConfig{})
	require.NoError(t, err)
	require.NotNil(t, rootDagExecution)

	rootTask, err := testSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{
		TaskId: rootDagExecution.TaskID,
	})
	require.NoError(t, err)
	parentTask := rootTask

	currentRun := &CurrentRun{
		Run:          run,
		ScopePath:    NewScopePath(pipelineSpec),
		T:            t,
		TestSetup:    testSetup,
		PipelineSpec: pipelineSpec,
	}
	err = currentRun.ScopePath.Push("root")
	require.NoError(t, err)

	// Run Dag on the First Task
	_, _ = currentRun.RunContainer("create-dataset", parentTask, nil)
	_, _ = currentRun.RunContainer("process-dataset", parentTask, nil)
	_, _ = currentRun.RunContainer("analyze-artifact", parentTask, nil)

}

func (r *CurrentRun) RunDag(
	taskName string,
	parentTask *apiv2beta1.PipelineTaskDetail) (*Execution, *apiv2beta1.PipelineTaskDetail) {
	t := r.T
	r.RefreshRun()
	err := r.ScopePath.Push(taskName)
	require.NoError(t, err)
	taskSpec := r.GetLast().GetTaskSpec()

	opts := setupDagOptions(t, r.TestSetup, r.Run, parentTask, taskSpec, r.PipelineSpec, nil)

	execution, err := DAG(context.Background(), opts, r.TestSetup.DriverAPI)
	require.NoError(t, err)
	require.NotNil(t, execution)

	task, err := r.TestSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: execution.TaskID})
	require.NoError(t, err)
	require.NotNil(t, task)
	require.Equal(t, execution.TaskID, task.TaskId)
	require.Equal(t, taskName, task.GetName())
	r.RefreshRun()
	return execution, task
}

func (r *CurrentRun) RunContainer(
	taskName string,
	parentTask *apiv2beta1.PipelineTaskDetail,
	iterationIndex *int64,
) (*Execution, *apiv2beta1.PipelineTaskDetail) {
	t := r.T
	r.RefreshRun()
	err := r.ScopePath.Push(taskName)
	require.NoError(t, err)
	taskSpec := r.GetLast().GetTaskSpec()

	opts := setupContainerOptions(t, r.TestSetup, r.Run, parentTask, taskSpec, r.PipelineSpec, nil)

	if iterationIndex != nil {
		opts.IterationIndex = int(*iterationIndex)
	}

	execution, err := Container(context.Background(), opts, r.TestSetup.DriverAPI)
	require.NoError(t, err)
	require.NotNil(t, execution)

	task, err := r.TestSetup.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: execution.TaskID})
	require.NoError(t, err)
	require.NotNil(t, task)
	require.Equal(t, execution.TaskID, task.TaskId)
	require.Equal(t, taskName, task.GetName())
	r.RefreshRun()
	_, ok := r.ScopePath.Pop()
	require.True(t, ok)
	return execution, task
}

func (r *CurrentRun) RefreshRun() {
	t := r.T
	run, err := r.TestSetup.DriverAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: r.Run.RunId})
	require.NoError(t, err)
	r.Run = run
}

func (r *CurrentRun) MockLauncherArtifactCreate(
	TaskId string,
	artifactKey string,
	artifactType apiv2beta1.Artifact_ArtifactType,
	outputType apiv2beta1.IOType,
	producerTask string,
	producerIteration *int64,
) string {
	t := r.T
	artifactID, _ := uuid.NewRandom()
	outputArtifact := &apiv2beta1.Artifact{
		ArtifactId: artifactID.String(),
		Name:       artifactKey,
		Type:       artifactType,
		Uri:        util.StringPointer(fmt.Sprintf("s3://some.location/%s", artifactKey)),
		Namespace:  testNamespace,
		Metadata: map[string]*structpb.Value{
			"display_name": structpb.NewStringValue(artifactKey),
		},
	}
	createArtifact, err := r.TestSetup.DriverAPI.CreateArtifact(
		context.Background(),
		&apiv2beta1.CreateArtifactRequest{
			Artifact: outputArtifact,
		})
	require.NoError(t, err)
	require.NotNil(t, createArtifact)

	artifactTask := &apiv2beta1.ArtifactTask{
		ArtifactId: artifactID.String(),
		TaskId:     TaskId,
		RunId:      r.Run.GetRunId(),
		Key:        artifactKey,
		Producer:   &apiv2beta1.IOProducer{TaskName: producerTask},
		Type:       outputType,
	}
	if producerIteration != nil {
		artifactTask.Producer.Iteration = producerIteration
	}
	at, err := r.TestSetup.DriverAPI.CreateArtifactTask(
		context.Background(),
		&apiv2beta1.CreateArtifactTaskRequest{
			ArtifactTask: artifactTask,
		})
	require.NoError(t, err)
	require.NotNil(t, at)
	r.RefreshRun()
	return artifactID.String()
}

func (r *CurrentRun) MockLauncherArtifactTaskCreate(
	producerTaskName, taskID, key string,
	artifactID string, producerIteration *int64,
	outputType apiv2beta1.IOType) {
	t := r.T
	at := &apiv2beta1.ArtifactTask{
		ArtifactId: artifactID,
		TaskId:     taskID,
		RunId:      r.Run.GetRunId(),
		Key:        key,
		Type:       outputType,
		Producer:   &apiv2beta1.IOProducer{TaskName: producerTaskName},
	}
	if producerIteration != nil {
		at.Producer.Iteration = producerIteration
	}
	result, err := r.TestSetup.DriverAPI.CreateArtifactTask(
		context.Background(),
		&apiv2beta1.CreateArtifactTaskRequest{ArtifactTask: at})
	require.NoError(t, err)
	require.NotNil(t, result)
	r.RefreshRun()
}
