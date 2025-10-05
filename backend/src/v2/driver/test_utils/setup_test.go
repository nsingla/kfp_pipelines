package test_utils

import (
	"context"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Example test demonstrating the usage including artifact population
func TestSetupTestSetup(t *testing.T) {
	// Setup test environment
	testSetup := NewTestSetup(t)

	// Create a test run
	run := testSetup.CreateTestRun(t, "test-pipeline")
	assert.NotNil(t, run)
	assert.NotEmpty(t, run.RunId)
	assert.Equal(t, "test-pipeline", run.GetPipelineSpec().Fields["pipelineInfo"].GetStructValue().Fields["name"].GetStringValue())

	// Create test tasks
	task1 := testSetup.CreateTestTask(t,
		run.RunId,
		"producer-task",
		apiv2beta1.PipelineTaskDetail_RUNTIME,
		[]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			CreateParameter(
				"input1",
				"pipelinechannel--args-generator-op-Output",
				&apiv2beta1.IOProducer{TaskName: "some-task"},
			),
		},
		[]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			CreateParameter(
				"output1",
				"msg",
				nil,
			),
			CreateParameter(
				"output2",
				"",
				nil,
			),
		})
	task2 := testSetup.CreateTestTask(t, run.RunId, "consumer-task", apiv2beta1.PipelineTaskDetail_RUNTIME,
		[]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			CreateParameter(
				"input4",
				"input4key",
				nil,
			),
		},
		[]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{
			CreateParameter(
				"output3",
				"pipelinechannel--split-ids-Output",
				nil,
			),
		})

	// Create test artifacts
	artifact1 := testSetup.CreateTestArtifact(t, "output-data", "dataset")
	artifact2 := testSetup.CreateTestArtifact(t, "trained-model", "model")

	// Create artifact-task relationships
	// task1 produces artifact1 (output)
	testSetup.CreateTestArtifactTask(t,
		artifact1.ArtifactId, task1.TaskId, run.RunId, "pipelinechannel--loop_parameter-loop-item-1",
		&apiv2beta1.IOProducer{
			TaskName: task1.Name,
		},
		apiv2beta1.IOType_OUTPUT,
	)

	// task2 consumes artifact1 (input)
	testSetup.CreateTestArtifactTask(t,
		artifact1.ArtifactId, task2.TaskId, run.RunId, "pipelinechannel--loop_parameter-loop-item-2",
		&apiv2beta1.IOProducer{
			TaskName: task1.Name,
		},
		apiv2beta1.IOType_COMPONENT_INPUT,
	)
	// task2 produces artifact2 (output)
	testSetup.CreateTestArtifactTask(t,
		artifact2.ArtifactId, task2.TaskId, run.RunId, "pipelinechannel--loop_parameter-loop-item",
		&apiv2beta1.IOProducer{
			TaskName: task2.Name,
		},
		apiv2beta1.IOType_OUTPUT,
	)

	// Test getting run with populated tasks and artifacts
	populatedRun, err := testSetup.DriverAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.RunId})
	require.NoError(t, err)
	assert.NotNil(t, populatedRun)
	assert.Len(t, populatedRun.Tasks, 2)

	// Verify task1 has correct artifacts (1 output)
	var producerTask *apiv2beta1.PipelineTaskDetail
	for _, task := range populatedRun.Tasks {
		if task.Name == "producer-task" {
			producerTask = task
			break
		}
	}
	require.NotNil(t, producerTask)
	assert.Len(t, producerTask.Inputs.Artifacts, 0)  // No input artifacts
	assert.Len(t, producerTask.Outputs.Artifacts, 1) // 1 output artifact

	// Verify task2 has correct artifacts (1 input, 1 output)
	var consumerTask *apiv2beta1.PipelineTaskDetail
	for _, task := range populatedRun.Tasks {
		if task.Name == "consumer-task" {
			consumerTask = task
			break
		}
	}
	require.NotNil(t, consumerTask)
	assert.Len(t, consumerTask.Inputs.Artifacts, 1)  // 1 input artifact
	assert.Len(t, consumerTask.Outputs.Artifacts, 1) // 1 output artifact

	// Verify producer information is correctly set
	inputArtifact := consumerTask.Inputs.Artifacts[0]
	assert.Equal(t, "producer-task", inputArtifact.GetProducer().TaskName)
	assert.Equal(t, "pipelinechannel--loop_parameter-loop-item-2", inputArtifact.GetArtifactKey())
}
