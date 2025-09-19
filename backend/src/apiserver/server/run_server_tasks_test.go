package server

import (
	"context"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// Helper to create a simple run via resource manager and return its ID.
func seedOneRun(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, string) {
	clients, manager, run := initWithOneTimeRunV2(t)
	return clients, manager, run.UUID
}

func TestTask_Create_Update_Get_List(t *testing.T) {
	// Single-user mode by default; keep it to bypass authz.
	clients, manager, runID := seedOneRun(t)
	defer clients.Close()

	runSrv := createRunServer(manager)

	// Create task with inputs/outputs
	inParams := []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{
		{Name: strPTR("p1"), Value: "v1"},
	}
	outParams := []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{
		{Name: strPTR("op1"), Value: "3.14"},
	}
	createReq := &apiv2beta1.CreateTaskRequest{Task: &apiv2beta1.PipelineTaskDetail{
		RunId:   runID,
		Name:    "trainer",
		Status:  apiv2beta1.RuntimeState_RUNNING,
		Type:    apiv2beta1.PipelineTaskDetail_RUNTIME,
		Inputs:  &apiv2beta1.PipelineTaskDetail_InputOutputs{Parameters: inParams},
		Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{Parameters: outParams},
	}}
	created, err := runSrv.CreateTask(context.Background(), createReq)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.GetTaskId())
	assert.Equal(t, runID, created.GetRunId())
	assert.Equal(t, "trainer", created.GetName())
	// Verify inputs/outputs echoed back
	assert.Len(t, created.GetInputs().GetParameters(), 1)
	assert.Equal(t, "p1", created.GetInputs().GetParameters()[0].GetName())
	assert.Len(t, created.GetOutputs().GetParameters(), 1)
	assert.Equal(t, "op1", created.GetOutputs().GetParameters()[0].GetName())

	// Update task: change status and outputs
	updReq := &apiv2beta1.UpdateTaskRequest{TaskId: created.GetTaskId(), Task: &apiv2beta1.PipelineTaskDetail{
		TaskId: created.GetTaskId(),
		RunId:  runID,
		Name:   "trainer",
		Status: apiv2beta1.RuntimeState_SUCCEEDED,
		Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{Parameters: []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{
			{Name: strPTR("op1"), Value: "done"},
		}},
	}}
	updated, err := runSrv.UpdateTask(context.Background(), updReq)
	assert.NoError(t, err)
	assert.Equal(t, apiv2beta1.RuntimeState_SUCCEEDED, updated.GetStatus())
	assert.Equal(t, "done", updated.GetOutputs().GetParameters()[0].GetValue())

	// GetTask
	got, err := runSrv.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: created.GetTaskId()})
	assert.NoError(t, err)
	assert.Equal(t, created.GetTaskId(), got.GetTaskId())
	assert.Equal(t, apiv2beta1.RuntimeState_SUCCEEDED, got.GetStatus())

	// ListTasks by run ID
	listResp, err := runSrv.ListTasks(context.Background(), &apiv2beta1.ListTasksRequest{ParentFilter: &apiv2beta1.ListTasksRequest_RunId{RunId: runID}, PageSize: 50})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, int(listResp.GetTotalSize()), 1)
	found := false
	for _, tt := range listResp.GetTasks() {
		if tt.GetTaskId() == created.GetTaskId() {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestTask_RunHydration_WithInputsOutputs_ArtifactsAndMetrics(t *testing.T) {
	// Multi-user on to exercise auth paths, but use helper ctx for headers.
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })

	clients, manager, run := initWithOneTimeRun(t)
	defer clients.Close()

	runSrv := createRunServer(manager)
	artSrv := createArtifactServer(manager)

	// Create a task with IO
	create := &apiv2beta1.CreateTaskRequest{
		Task: &apiv2beta1.PipelineTaskDetail{
			RunId:  run.UUID,
			Name:   "preprocess",
			Status: apiv2beta1.RuntimeState_RUNNING,
			Inputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{Parameters: []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{
				{Name: strPTR("threshold"), Value: "0.5"},
			}},
			Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{Parameters: []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{
				{Name: strPTR("rows"), Value: "100"},
			}},
		}}
	created, err := runSrv.CreateTask(ctxWithUser(), create)
	assert.NoError(t, err)

	// Create an artifact and link it as output of the task
	_, err = artSrv.CreateArtifact(ctxWithUser(),
		&apiv2beta1.CreateArtifactRequest{
			RunId:       run.UUID,
			TaskId:      created.GetTaskId(),
			ProducerKey: "some-parent-task-output",
			Artifact: &apiv2beta1.Artifact{
				Namespace: run.Namespace,
				Type:      apiv2beta1.Artifact_Model,
				Uri:       strPTR("gs://bucket/model"),
				Name:      "m1",
			}})
	assert.NoError(t, err)

	// Confirm a link was created between the task and the artifact
	artifactTasks, err := artSrv.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{
		TaskIds:  []string{created.GetTaskId()},
		RunIds:   []string{run.UUID},
		PageSize: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), artifactTasks.GetTotalSize())
	assert.Equal(t, 1, len(artifactTasks.GetArtifactTasks()))

	// Update task outputs to include an artifact reference in OutputArtifacts
	_, err = runSrv.UpdateTask(ctxWithUser(),
		&apiv2beta1.UpdateTaskRequest{
			TaskId: created.GetTaskId(),
			Task: &apiv2beta1.PipelineTaskDetail{
				TaskId:  created.GetTaskId(),
				RunId:   run.UUID,
				Status:  apiv2beta1.RuntimeState_SUCCEEDED,
				Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{},
			}})
	assert.NoError(t, err)

	// Now fetch the run and ensure tasks are hydrated with inputs/outputs
	gr, err := runSrv.GetRun(ctxWithUser(), &apiv2beta1.GetRunRequest{RunId: run.UUID})
	assert.NoError(t, err)
	assert.NotNil(t, gr)
	assert.GreaterOrEqual(t, len(gr.GetTasks()), 1)
	var taskFound *apiv2beta1.PipelineTaskDetail
	for _, tt := range gr.GetTasks() {
		if tt.GetTaskId() == created.GetTaskId() {
			taskFound = tt
			break
		}
	}
	if assert.NotNil(t, taskFound, "created task not present in hydrated run") {
		// Parameters present
		assert.Equal(t, 1, len(taskFound.GetInputs().GetParameters()))
		if assert.NotNil(t, taskFound.GetInputs().GetParameters(), "parameters not present in hydrated task") {
			assert.Equal(t, "threshold", taskFound.GetInputs().GetParameters()[0].GetName())
			// Outputs updated and artifact reference present
			assert.Equal(t, apiv2beta1.RuntimeState_SUCCEEDED, taskFound.GetStatus())
		}
		assert.Equal(t, 1, len(taskFound.GetOutputs().GetArtifacts()))
		if assert.NotNil(t, taskFound.GetOutputs().GetArtifacts(), "artifacts not present in hydrated task") {
			assert.Equal(t, taskFound.Name, taskFound.GetOutputs().GetArtifacts()[0].GetProducer().GetTaskName())
			assert.Equal(t, "some-parent-task-output", taskFound.GetOutputs().GetArtifacts()[0].GetProducer().Key)
			assert.Equal(t, "gs://bucket/model", *taskFound.GetOutputs().GetArtifacts()[0].GetValue().Uri)
			assert.Equal(t, "m1", taskFound.GetOutputs().GetArtifacts()[0].GetValue().Name)
		}
	}

}

func TestListTasks_ByParent(t *testing.T) {
	cm, rm, runID := seedOneRun(t)
	defer cm.Close()
	server := createRunServer(rm)

	// Create parent task
	parent, err := server.CreateTask(context.Background(),
		&apiv2beta1.CreateTaskRequest{
			Task: &apiv2beta1.PipelineTaskDetail{
				RunId: runID,
				Name:  "parent",
			},
		},
	)
	assert.NoError(t, err)

	// Create child task with ParentTaskId
	child, err := server.CreateTask(context.Background(),
		&apiv2beta1.CreateTaskRequest{
			Task: &apiv2beta1.PipelineTaskDetail{
				RunId:        runID,
				Name:         "child",
				ParentTaskId: parent.GetTaskId(),
			},
		},
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, child.GetTaskId())

	// List by parent ID
	resp, err := server.ListTasks(context.Background(),
		&apiv2beta1.ListTasksRequest{
			ParentFilter: &apiv2beta1.ListTasksRequest_ParentId{
				ParentId: parent.GetTaskId(),
			},
			PageSize: 50,
		})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.GetTotalSize())
	assert.Equal(t, 1, len(resp.GetTasks()))
	assert.Equal(t, child.GetTaskId(), resp.GetTasks()[0].GetTaskId())
}
