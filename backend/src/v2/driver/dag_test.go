package driver

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestLoopArtifactPassing(t *testing.T) {
	tc := NewTestContext(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/loop_collected.py.yaml")
	parentTask := tc.RootTask

	// Run Dag on the First Task
	secondaryPipelineExecution, secondaryPipelineTask := tc.RunDag("secondary-pipeline", parentTask)
	require.Nil(t, secondaryPipelineExecution.ExecutorInput.Outputs)
	require.Equal(t, apiv2beta1.PipelineTaskDetail_RUNNING, secondaryPipelineTask.Status)

	// Refresh Parent Task - The parent task should be the secondary pipeline task for "create-dataset"
	parentTask = secondaryPipelineTask

	// Now we'll run the subtasks in the secondary pipeline, one of which is a loop of 3 iterations

	// Run the Downstream Task that will use the output artifact
	createDataSetExecution, _ := tc.RunContainer("create-dataset", parentTask, nil)
	require.Nil(t, createDataSetExecution.ExecutorInput.Outputs)

	// Mock a Launcher run by updating the task with output data
	createDataSetOutputArtifactID := tc.MockLauncherArtifactCreate(
		createDataSetExecution.TaskID,
		"output_dataset",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		"create-dataset",
		nil,
	)

	// Run the Loop Task - note that parentTask for for-loop-2 remains as secondary-pipeline
	loopExecution, loopTask := tc.RunDag("for-loop-2", parentTask)
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
		processExecution, _ := tc.RunContainer("process-dataset", parentTask, util.Int64Pointer(int64(index)))
		require.Nil(t, processExecution.ExecutorInput.Outputs)
		require.NotNil(t, processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"])
		require.Equal(t, 1, len(processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"].GetArtifacts()))
		require.Equal(t, processExecution.ExecutorInput.Inputs.Artifacts["input_dataset"].GetArtifacts()[0].ArtifactId, createDataSetOutputArtifactID)
		require.NotNil(t, processExecution.ExecutorInput.Inputs.ParameterValues["model_id_in"])
		require.Equal(t, processExecution.ExecutorInput.Inputs.ParameterValues["model_id_in"].GetStringValue(), paramID)

		// Mock the Launcher run
		processDataSetArtifactID := tc.MockLauncherArtifactCreate(
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
		tc.MockLauncherArtifactTaskCreate(
			"process-dataset",
			loopExecution.TaskID,
			"pipelinechannel--process-dataset-output_artifact",
			processDataSetArtifactID,
			util.Int64Pointer(int64(index)),
			apiv2beta1.IOType_ITERATOR_OUTPUT,
		)
		loopTask, err := tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: loopExecution.TaskID})
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
		tc.MockLauncherArtifactTaskCreate(
			"process-dataset",
			secondaryPipelineExecution.TaskID,
			"Output",
			processDataSetArtifactID,
			util.Int64Pointer(int64(index)),
			apiv2beta1.IOType_ITERATOR_OUTPUT,
		)
		secondaryPipelineTask, err = tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: secondaryPipelineExecution.TaskID})
		require.NoError(t, err)
		require.NotNil(t, secondaryPipelineTask.Outputs)
		require.Equal(t, len(secondaryPipelineTask.Outputs.Artifacts), index+1)

		// Run next iteration component
		analyzeExecution, _ := tc.RunContainer("analyze-artifact", parentTask, util.Int64Pointer(int64(index)))
		require.Nil(t, createDataSetExecution.ExecutorInput.Outputs)
		require.Nil(t, analyzeExecution.ExecutorInput.Outputs)
		require.NotNil(t, analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"])
		require.Equal(t, 1, len(analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"].GetArtifacts()))
		require.Equal(t, analyzeExecution.ExecutorInput.Inputs.Artifacts["analyze_artifact_input"].GetArtifacts()[0].ArtifactId, processDataSetArtifactID)

		// Mock the Launcher run
		_ = tc.MockLauncherArtifactCreate(
			processExecution.TaskID,
			"analyze_output_artifact",
			apiv2beta1.Artifact_Artifact,
			apiv2beta1.IOType_OUTPUT,
			"analyze-artifact",
			util.Int64Pointer(int64(index)),
		)
	}

	tasks, err := tc.DriverAPI.ListTasks(context.Background(), &apiv2beta1.ListTasksRequest{
		ParentFilter: &apiv2beta1.ListTasksRequest_ParentId{ParentId: loopExecution.TaskID},
	})
	require.NoError(t, err)
	require.NotNil(t, tasks)
	// Expect 3 tasks for analyze-artifact + 3 tasks for process-dataset
	require.Equal(t, 6, len(tasks.Tasks))

	// Expect the 3 artifacts from process-task to have been collected by the for-loop-2 task
	forLoopTask, err := tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: loopExecution.TaskID})
	require.NoError(t, err)
	require.Equal(t, 3, len(forLoopTask.Outputs.Artifacts))

	// Run "analyze_artifact_list" in "secondary_pipeline"

	// Move up a parent
	parentTask = secondaryPipelineTask
	_, ok := tc.ScopePath.Pop()
	require.True(t, ok)

	analyzeArtifactListExecution, _ := tc.RunContainer("analyze-artifact-list", parentTask, nil)
	require.Nil(t, analyzeArtifactListExecution.ExecutorInput.Outputs)
	require.NotNil(t, analyzeArtifactListExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"])
	require.Equal(t, 3, len(analyzeArtifactListExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"].GetArtifacts()))

	// Primary Pipeline tests

	// Expect the 3 artifacts from process-task to have been collected by the secondary-pipeline task
	secondaryPipelineTask, err = tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: secondaryPipelineExecution.TaskID})
	require.NoError(t, err)
	require.Equal(t, 3, len(secondaryPipelineTask.Outputs.Artifacts))

	// Move up a parent
	parentTask = tc.RootTask
	_, ok = tc.ScopePath.Pop()
	require.True(t, ok)

	// Not to be confused with the "analyze-artifact-list" task in secondary pipeline,
	// this is the "analyze-artifact-list" task in the primary pipeline
	analyzeArtifactListOuterExecution, _ := tc.RunContainer("analyze-artifact-list", parentTask, nil)
	require.Nil(t, analyzeArtifactListExecution.ExecutorInput.Outputs)
	require.Nil(t, analyzeArtifactListOuterExecution.ExecutorInput.Outputs)
	require.NotNil(t, analyzeArtifactListOuterExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"])
	require.Equal(t, 3, len(analyzeArtifactListOuterExecution.ExecutorInput.Inputs.Artifacts["artifact_list_input"].GetArtifacts()))

	// Refresh Run so it has the new tasks
	tc.RefreshRun()

	// primary_pipeline()		 x 1  (root)
	// secondary_pipeline()      x 1  (dag)
	//   create_dataset()        x 1  (runtime)
	//   for_loop_1()            x 1  (loop)
	//     process_dataset()     x 3  (runtime)
	//	   analyze_artifact()    x 3  (runtime)
	//   analyze_artifact_list() x 1  (runtime)
	// analyze_artifact_list()   x 1  (runtime)
	require.Equal(t, 12, len(tc.Run.Tasks))
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
	tc := NewTestContext(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/taskOutputParameter_test.py.yaml")
	parentTask := tc.RootTask

	// Run Dag on the First Task
	cdExecution, _ := tc.RunContainer("create-dataset", parentTask, nil)
	tc.MockLauncherParameterCreate(
		cdExecution.TaskID,
		"output_parameter_path",
		&structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: 10.0}},
		apiv2beta1.IOType_OUTPUT,
		"create-dataset",
		nil,
	)
	pdExecution, _ := tc.RunContainer("process-dataset", parentTask, nil)
	tc.MockLauncherParameterCreate(
		pdExecution.TaskID,
		"output_int",
		&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "output_int_value"}},
		apiv2beta1.IOType_OUTPUT,
		"process-dataset",
		nil,
	)
	analyzeArtifactExecution, _ := tc.RunContainer("analyze-artifact", parentTask, nil)
	tc.MockLauncherParameterCreate(
		analyzeArtifactExecution.TaskID,
		"output_opinion",
		&structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: true}},
		apiv2beta1.IOType_OUTPUT,
		"analyze-artifact",
		nil,
	)
}
