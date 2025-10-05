package driver

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRootDagComponentInputs(t *testing.T) {
	runtimeConfig := &pipelinespec.PipelineJob_RuntimeConfig{
		ParameterValues: map[string]*structpb.Value{
			"string_input": structpb.NewStringValue("test-input1"),
			"number_input": structpb.NewNumberValue(42.5),
			"bool_input":   structpb.NewBoolValue(true),
			"null_input":   structpb.NewNullValue(),
			"list_input": structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{
				structpb.NewStringValue("value1"),
				structpb.NewNumberValue(42),
				structpb.NewBoolValue(true),
			}}),
			"map_input": structpb.NewStructValue(&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"key1": structpb.NewStringValue("value1"),
					"key2": structpb.NewNumberValue(42),
					"key3": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{
							structpb.NewStringValue("nested1"),
							structpb.NewStringValue("nested2"),
						},
					}),
				},
			}),
		},
	}

	tc := NewTestContextWithRootExecuted(t, runtimeConfig, "test_data/taskOutputArtifact_test.py.yaml")
	task := tc.RootTask
	require.NotNil(t, task.Inputs)
	require.NotEmpty(t, task.Inputs.Parameters)

	// Verify parameter values
	paramMap := make(map[string]*structpb.Value)
	for _, param := range task.Inputs.Parameters {
		paramMap[param.GetParameterKey()] = param.Value
	}

	assert.Equal(t, "test-input1", paramMap["string_input"].GetStringValue())
	assert.Equal(t, 42.5, paramMap["number_input"].GetNumberValue())
	assert.Equal(t, true, paramMap["bool_input"].GetBoolValue())
	assert.NotNil(t, paramMap["null_input"].GetNullValue())
	assert.Len(t, paramMap["list_input"].GetListValue().Values, 3)
	assert.NotNil(t, paramMap["map_input"].GetStructValue())
	assert.Len(t, paramMap["map_input"].GetStructValue().Fields, 3)
}

func TestLoopArtifactPassing(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/loop_collected.py.yaml")
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
	tc.ExitDag()

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
	tc.ExitDag()

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

// TestParameterInputIterator will test parameter Input Iterator
// and parameter collection from output of a task in a loop
func TestParameterInputIterator(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/loop_collected_dynamic.py.yaml")
	// Execute full pipeline
	parentTask := tc.RootTask
	_, secondaryPipelineTask := tc.RunDag("secondary-pipeline", parentTask)
	parentTask = secondaryPipelineTask
	_, splitIDsTask := tc.RunContainer("split-ids", parentTask, nil)

	tc.MockLauncherParameterCreate(
		splitIDsTask.GetTaskId(),
		"Output",
		&structpb.Value{
			Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{
				Values: []*structpb.Value{
					structpb.NewStringValue("1"),
					structpb.NewStringValue("2"),
					structpb.NewStringValue("3"),
				},
			},
			},
		},
		apiv2beta1.IOType_OUTPUT,
		"split-ids",
		nil,
	)

	_, loopTask := tc.RunDag("for-loop-1", parentTask)
	parentTask = loopTask

	for index, _ := range []string{"1", "2", "3"} {
		index64 := util.Int64Pointer(int64(index))
		_, createFileTask := tc.RunContainer(
			"create-file",
			parentTask,
			index64,
		)

		tc.MockLauncherArtifactCreate(
			createFileTask.GetTaskId(),
			"file",
			apiv2beta1.Artifact_Artifact,
			apiv2beta1.IOType_OUTPUT,
			"create-file",
			index64,
		)

		// Run next task
		_, readSingleFileTask := tc.RunContainer(
			"read-single-file",
			parentTask,
			index64,
		)
		mockSingleFileTaskOutputParameterValue := &structpb.Value{
			Kind: &structpb.Value_StringValue{
				StringValue: fmt.Sprintf("file-%d", index),
			},
		}
		tc.MockLauncherParameterCreate(
			readSingleFileTask.GetTaskId(),
			"Output",
			mockSingleFileTaskOutputParameterValue,
			apiv2beta1.IOType_ITERATOR_OUTPUT,
			"read-single-file",
			index64,
		)

		// Parameter should be also sent upstream for collection
		tc.MockLauncherParameterCreate(
			loopTask.GetTaskId(),
			"pipelinechannel--read-single-file-Output",
			mockSingleFileTaskOutputParameterValue,
			apiv2beta1.IOType_ITERATOR_OUTPUT,
			"read-single-file",
			index64,
		)
		tc.MockLauncherParameterCreate(
			secondaryPipelineTask.GetTaskId(),
			"Output",
			mockSingleFileTaskOutputParameterValue,
			apiv2beta1.IOType_ITERATOR_OUTPUT,
			"read-single-file",
			index64,
		)

	}

	tc.ExitDag()
	parentTask = secondaryPipelineTask

	_, readValuesTask := tc.RunContainer("read-values", parentTask, nil)
	tc.MockLauncherParameterCreate(
		readValuesTask.GetTaskId(),
		"Output",
		&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "files read"}},
		apiv2beta1.IOType_OUTPUT,
		"read-values",
		nil,
	)

	tc.ExitDag()
	parentTask = tc.RootTask

	_, readValuesTask2 := tc.RunContainer("read-values", parentTask, nil)
	tc.MockLauncherParameterCreate(
		readValuesTask2.GetTaskId(),
		"Output",
		&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "files read"}},
		apiv2beta1.IOType_OUTPUT,
		"read-values",
		nil,
	)

	task, err := tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: secondaryPipelineTask.GetTaskId()})
	require.NoError(t, err)
	require.NotNil(t, task.Outputs)
	require.Equal(t, 3, len(task.Outputs.Parameters))
	var collectOutputs []string
	for _, params := range task.Outputs.Parameters {
		collectOutputs = append(collectOutputs, params.GetValue().GetStringValue())
		require.Equal(t, apiv2beta1.IOType_ITERATOR_OUTPUT, params.GetType())
	}
	require.Equal(t, []string{"file-0", "file-1", "file-2"}, collectOutputs)
}

func TestNestedDag(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/nested_naming_conflicts.py.yaml")
	parentTask := tc.RootTask

	_, aTask := tc.RunContainer("a", parentTask, nil)
	tc.MockLauncherArtifactCreate(
		aTask.GetTaskId(),
		"output_dataset",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		"a",
		nil)

	_, pipelineBTask := tc.RunDag("pipeline-b", parentTask)
	parentTask = pipelineBTask

	_, nestedATask := tc.RunContainer("a", parentTask, nil)
	tc.MockLauncherArtifactCreate(
		nestedATask.GetTaskId(),
		"output_dataset",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		"a",
		nil)

	_, nestedBTask := tc.RunContainer("b", parentTask, nil)
	tc.MockLauncherArtifactCreate(
		nestedBTask.GetTaskId(),
		"output_artifact_b",
		apiv2beta1.Artifact_Artifact,
		apiv2beta1.IOType_OUTPUT,
		"b",
		nil)

	_, pipelineCTask := tc.RunDag("pipeline-c", parentTask)
	parentTask = pipelineCTask

	_, nestedNestedATask := tc.RunContainer("a", parentTask, nil)
	tc.MockLauncherArtifactCreate(
		nestedNestedATask.GetTaskId(),
		"output_dataset",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		"a",
		nil)

	_, nestedNestedBTask := tc.RunContainer("b", parentTask, nil)
	tc.MockLauncherArtifactCreate(
		nestedNestedBTask.GetTaskId(),
		"output_artifact_b",
		apiv2beta1.Artifact_Artifact,
		apiv2beta1.IOType_OUTPUT,
		"b",
		nil)

	_, cTask := tc.RunContainer("c", parentTask, nil)
	cTaskArtifactID := tc.MockLauncherArtifactCreate(
		cTask.GetTaskId(),
		"output_artifact_c",
		apiv2beta1.Artifact_Artifact,
		apiv2beta1.IOType_OUTPUT,
		"c",
		nil)

	// Dag output for pipeline_b
	tc.MockLauncherArtifactTaskCreate(
		cTask.GetName(),
		pipelineBTask.GetTaskId(),
		"Output",
		cTaskArtifactID,
		nil,
		apiv2beta1.IOType_OUTPUT,
	)

	tc.ExitDag()
	parentTask = pipelineBTask

	tc.ExitDag()
	parentTask = tc.RootTask

	_, _ = tc.RunContainer("verify", parentTask, nil)

	var err error

	// Confirm that the artifact passed to "verify" task came from task_c
	pipelineBTask, err = tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: pipelineBTask.GetTaskId()})
	require.NoError(t, err)
	require.NotNil(t, pipelineBTask.Outputs)
	require.Equal(t, 1, len(pipelineBTask.Outputs.Artifacts))
	require.Equal(t, cTask.GetTaskId(), pipelineBTask.Outputs.Artifacts[0].GetArtifacts()[0].GetMetadata()["task_id"].GetStringValue())

	// Confirm that the artifact passed to cTask came from the nestedNestedBtask
	// I.e the b() task that ran in pipeline-c and not in pipeline-b
	cTask, err = tc.DriverAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: cTask.GetTaskId()})
	require.NoError(t, err)
	require.NotNil(t, cTask.Outputs)
	require.Equal(t, 1, len(cTask.Outputs.Artifacts))
	require.Equal(t, nestedNestedBTask.GetTaskId(), cTask.Inputs.Artifacts[0].GetArtifacts()[0].GetMetadata()["task_id"].GetStringValue())
}

func TestParameterTaskOutput(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/taskOutputParameter_test.py.yaml")
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

// TODO
func TestOneOf(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/oneof_simple.yaml")
	parentTask := tc.RootTask
	require.NotNil(t, parentTask)

	// Run secondary pipeline
	_, secondaryPipelineTask := tc.RunDag("secondary-pipeline", parentTask)
	parentTask = secondaryPipelineTask

	// Run create_dataset()
	_, createDatasetTask := tc.RunContainer("create-dataset", parentTask, nil)
	tc.MockLauncherArtifactCreate(
		createDatasetTask.GetTaskId(),
		"output_dataset",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		createDatasetTask.GetName(),
		nil,
	)
	tc.MockLauncherParameterCreate(
		createDatasetTask.GetTaskId(),
		"condition_out",
		&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "second"}},
		apiv2beta1.IOType_OUTPUT,
		createDatasetTask.GetName(),
		nil,
	)

	// Run ConditionBranch
	_, conditionBranch1Task := tc.RunDag("condition-branches-1", parentTask)
	parentTask = conditionBranch1Task

	// Expect this condition to not be met
	condition2Execution, _ := tc.RunDag("condition-2", conditionBranch1Task)
	require.NotNil(t, condition2Execution.Condition)
	require.False(t, *condition2Execution.Condition)

	tc.ExitDag()

	// Expect this condition to pass since output of
	// create-dataset == "second"
	condition3Execution, _ := tc.RunDag("condition-3", conditionBranch1Task)
	require.NotNil(t, condition3Execution.Condition)
	require.True(t, *condition3Execution.Condition)

	tc.ExitDag()

	// Expect this condition to not be met
	condition4Execution, _ := tc.RunDag("condition-4", conditionBranch1Task)
	require.NotNil(t, condition4Execution.Condition)
	require.False(t, *condition4Execution.Condition)

	tc.ExitDag()
}

// TODO: Probably covered by oneOF
func TestConditions(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/conditions_level_1_test.py.yaml")
	parentTask := tc.RootTask
	require.NotNil(t, parentTask)

}

func TestFinalStatus(t *testing.T) {
}

func TestOptionalFields(t *testing.T) {}

func TestWithCaching(t *testing.T) {
	// TODO Can't do this unless we can mock the filter API call on fingerprint and status
	// See getFingerPrintsANDID
}

func TestArtifactIterator(t *testing.T) {

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

	tc := NewTestContextWithRootExecuted(t, runtimeInputs, "test_data/componentInput_level_1_test.py.yaml")

	// Run Container on the First Task
	processInputsExecution, processInputsTask := tc.RunContainer("process-inputs", tc.RootTask, nil)
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
	tc.MockLauncherArtifactCreate(
		processInputsTask.TaskId,
		"output_text",
		apiv2beta1.Artifact_Dataset,
		apiv2beta1.IOType_OUTPUT,
		"process-inputs",
		nil,
	)

	analyzeInputsExecution, _ := tc.RunContainer("analyze-inputs", tc.RootTask, nil)
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

func fetchParameter(key string, params []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter) *apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter {
	for _, p := range params {
		if key == p.ParameterKey {
			return p
		}
	}
	panic("parameter not found")
}
