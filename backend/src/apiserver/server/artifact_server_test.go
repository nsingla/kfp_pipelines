package server

import (
	"context"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

const (
	runid1 = "123e4567-e89b-12d3-a456-426655440001"
)

func createArtifactServer(resourceManager *resource.ResourceManager) *ArtifactServer {
	return &ArtifactServer{resourceManager: resourceManager}
}

// ctxWithUser returns a context with a fake user identity header so that
// authorization in multi-user mode passes in tests.
func ctxWithUser() context.Context {
	header := common.GetKubeflowUserIDHeader()
	prefix := common.GetKubeflowUserIDPrefix()
	// Typical header value is like: "accounts.google.com:alice@example.com"
	val := prefix + "test-user@example.com"
	md := metadata.New(map[string]string{header: val})
	return metadata.NewIncomingContext(context.Background(), md)
}

func strPTR(s string) *string {
	return &s
}

func TestArtifactServer_CreateArtifact_MultiUserCreateAndGet_Succeeds(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Create run
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         runid1,
		K8SName:      "test-run",
		DisplayName:  "test-run",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			State:            model.RuntimeStateRunning,
		},
	})
	assert.NoError(t, err)

	// Create task for the run
	task, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace:    "ns1",
		PipelineName: "test-pipeline",
		RunUUID:      runid1,
		Name:         "test-task",
		Status:       1,
	})
	assert.NoError(t, err)

	req := &apiv2beta1.CreateArtifactRequest{
		RunId:       runid1,
		TaskId:      task.UUID,
		ProducerKey: "producer-key",
		Artifact: &apiv2beta1.Artifact{
			Namespace:   "ns1",
			Type:        apiv2beta1.Artifact_Model,
			Uri:         strPTR("gs://b/f"),
			Name:        "a1",
			Description: "desc1",
		}}
	created, err := s.CreateArtifact(ctxWithUser(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.GetArtifactId())
	assert.Equal(t, "ns1", created.GetNamespace())
	assert.Equal(t, apiv2beta1.Artifact_Model, created.GetType())
	assert.Equal(t, "gs://b/f", created.GetUri())
	assert.Equal(t, "a1", created.GetName())
	assert.Equal(t, "desc1", created.GetDescription())

	// Creating an artifact should create an artifact task
	// Fetch the artifact task
	artifactTasks, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{
		TaskIds:  []string{task.UUID},
		PageSize: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), artifactTasks.GetTotalSize())
	assert.Equal(t, 1, len(artifactTasks.GetArtifactTasks()))

	at := artifactTasks.GetArtifactTasks()[0]
	assert.Equal(t, created.GetArtifactId(), at.GetArtifactId())
	assert.Equal(t, task.UUID, at.GetTaskId())
	assert.Equal(t, apiv2beta1.ArtifactTaskType_OUTPUT, at.GetType())
	assert.Equal(t, task.Name, at.GetProducerTaskName())
	assert.Equal(t, "producer-key", at.GetProducerKey())
	// ArtifactKey may be empty for CreateArtifact path; ensure it defaults to empty string
	assert.Equal(t, "", at.GetArtifactKey())

}

func TestArtifactServer_ListArtifacts_HappyPath(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Create required run and task for artifact creation
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         runid1,
		K8SName:      "list-run",
		DisplayName:  "list-run",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails:   model.RunDetails{CreatedAtInSec: 1, ScheduledAtInSec: 1, State: model.RuntimeStateRunning},
	})
	assert.NoError(t, err)
	listTask, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace:    "ns1",
		PipelineName: "p-list",
		RunUUID:      runid1,
		Name:         "t-list",
		Status:       1,
	})
	assert.NoError(t, err)
	_, err = s.CreateArtifact(ctxWithUser(), &apiv2beta1.CreateArtifactRequest{RunId: runid1, TaskId: listTask.UUID, ProducerKey: "producer-key", Artifact: &apiv2beta1.Artifact{
		Namespace:   "ns1",
		Type:        apiv2beta1.Artifact_Model,
		Uri:         strPTR("gs://b/f"),
		Name:        "a1",
		Description: "desc-list",
	}})
	listResp, err := s.ListArtifacts(ctxWithUser(), &apiv2beta1.ListArtifactRequest{
		Namespace: "ns1",
		PageSize:  10,
	})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, int(listResp.GetTotalSize()), 1)
	assert.GreaterOrEqual(t, len(listResp.GetArtifacts()), 1)
}

func TestArtifactServer_GetArtifact_Errors(t *testing.T) {
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Missing ID
	_, err := s.GetArtifact(context.Background(), &apiv2beta1.GetArtifactRequest{ArtifactId: ""})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())

	// Non-existent
	_, err = s.GetArtifact(context.Background(), &apiv2beta1.GetArtifactRequest{ArtifactId: "does-not-exist"})
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestArtifactServer_Authorization_MultiUser(t *testing.T) {
	// Turn on MU mode by setting viper flag
	// Note: IsMultiUserMode() reads from viper, so configure it here
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	// In MU, Create should preserve namespace; and List with empty namespace should fail
	s := createArtifactServer(resourceManager)

	// By default FakeResourceManager authorizes everything in MU, unless namespace is empty
	// ListArtifacts with empty namespace should fail in MU
	_, err := s.ListArtifacts(ctxWithUser(), &apiv2beta1.ListArtifactRequest{Namespace: ""})
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestArtifactServer_SingleUserNamespaceEmpty(t *testing.T) {
	// Ensure single-user mode
	viper.Set(common.MultiUserMode, "false")
	clientManager := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Even if request carries a namespace, in single-user mode it should be cleared/empty in stored artifact
	// Create run and task required by CreateArtifact
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         "single-run",
		K8SName:      "single-run",
		DisplayName:  "single-run",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails:   model.RunDetails{CreatedAtInSec: 1, ScheduledAtInSec: 1, State: model.RuntimeStateRunning},
	})
	assert.NoError(t, err)
	singleTask, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace:    "ns1",
		PipelineName: "p-single",
		RunUUID:      "single-run",
		Name:         "t-single",
		Status:       1,
	})
	assert.NoError(t, err)
	created, err := s.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		RunId:       "single-run",
		TaskId:      singleTask.UUID,
		ProducerKey: "producer-key",
		Artifact: &apiv2beta1.Artifact{
			Namespace:   "ns1",
			Type:        apiv2beta1.Artifact_Artifact,
			Uri:         strPTR("u"),
			Name:        "a",
			Description: "single-desc",
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "", created.GetNamespace())

	// Get artifact and verify it matches
	fetched, err := s.GetArtifact(context.Background(), &apiv2beta1.GetArtifactRequest{
		ArtifactId: created.GetArtifactId(),
	})
	assert.NoError(t, err)
	assert.Equal(t, "", fetched.GetNamespace())
	assert.Equal(t, apiv2beta1.Artifact_Artifact, fetched.GetType())
	assert.Equal(t, "u", fetched.GetUri())
	assert.Equal(t, "a", fetched.GetName())
	assert.Equal(t, "single-desc", fetched.GetDescription())
}

const (
	serverRunID1 = "run-1"
	serverRunID2 = "run-2"
)

// seedArtifactTasks sets up two runs, two tasks, two artifacts and three links.
// Returns server, clientManager, entities.
func seedArtifactTasks(t *testing.T) (*ArtifactServer, *resource.FakeClientManager, *model.Task, *model.Task, *model.Artifact, *model.Artifact) {
	viper.Set(common.MultiUserMode, "true")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false") })
	clientManager := resource.NewFakeClientManagerOrFatalV2()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	s := createArtifactServer(resourceManager)

	// Runs
	_, err := clientManager.RunStore().CreateRun(&model.Run{
		UUID:         serverRunID1,
		ExperimentId: "",
		K8SName:      "r1",
		DisplayName:  "r1",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			State:            model.RuntimeStateRunning,
		},
	})
	assert.NoError(t, err)
	_, err = clientManager.RunStore().CreateRun(&model.Run{
		UUID:         serverRunID2,
		ExperimentId: "",
		K8SName:      "r2",
		DisplayName:  "r2",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			State:            model.RuntimeStateRunning,
		},
	})

	// Tasks
	t1, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace:    "ns1",
		PipelineName: "p1",
		RunUUID:      serverRunID1,
		Name:         "t1",
		Status:       1,
	})
	assert.NoError(t, err)
	t2, err := clientManager.TaskStore().CreateTask(&model.Task{
		Namespace:    "ns1",
		PipelineName: "p1",
		RunUUID:      serverRunID2,
		Name:         "t2",
		Status:       1,
	})
	assert.NoError(t, err)

	// Artifacts
	art1, err := clientManager.ArtifactStore().CreateArtifact(&model.Artifact{
		Namespace:   "ns1",
		Type:        model.ArtifactType(apiv2beta1.Artifact_Artifact),
		Uri:         strPTR("u"),
		Name:        "a1",
		Description: "d1",
	})
	assert.NoError(t, err)
	art2, err := clientManager.ArtifactStore().CreateArtifact(&model.Artifact{
		Namespace:   "ns1",
		Type:        model.ArtifactType(apiv2beta1.Artifact_Artifact),
		Uri:         strPTR("u2"),
		Name:        "a2",
		Description: "d2",
	})
	assert.NoError(t, err)

	// Links
	_, err = s.CreateArtifactTask(ctxWithUser(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: &apiv2beta1.ArtifactTask{
			ArtifactId:       art1.UUID,
			TaskId:           t1.UUID,
			RunId:            serverRunID1,
			Type:             apiv2beta1.ArtifactTaskType_INPUT,
			ProducerTaskName: "t1",
			ProducerKey:      "k1",
			ArtifactKey:      "in1",
		},
	})
	assert.NoError(t, err)
	_, err = s.CreateArtifactTask(ctxWithUser(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: &apiv2beta1.ArtifactTask{
			ArtifactId:       art2.UUID,
			TaskId:           t1.UUID,
			RunId:            serverRunID1,
			Type:             apiv2beta1.ArtifactTaskType_OUTPUT,
			ProducerTaskName: "t1",
			ProducerKey:      "k2",
			ArtifactKey:      "out1",
		},
	})
	assert.NoError(t, err)
	_, err = s.CreateArtifactTask(ctxWithUser(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: &apiv2beta1.ArtifactTask{
			ArtifactId:       art2.UUID,
			TaskId:           t2.UUID,
			RunId:            serverRunID2,
			Type:             apiv2beta1.ArtifactTaskType_INPUT,
			ProducerTaskName: "t2",
			ProducerKey:      "k3",
			ArtifactKey:      "in2",
		},
	})
	assert.NoError(t, err)

	return s, clientManager, t1, t2, art1, art2
}

func TestArtifactServer_ListArtifactTasks_FilterByTaskIds(t *testing.T) {
	s, _, t1, _, _, _ := seedArtifactTasks(t)
	resp, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{TaskIds: []string{t1.UUID}, PageSize: 50})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.GetTotalSize())
	assert.Equal(t, 2, len(resp.GetArtifactTasks()))
	// Ensure artifact_key values are present
	keys := []string{resp.GetArtifactTasks()[0].GetArtifactKey(), resp.GetArtifactTasks()[1].GetArtifactKey()}
	assert.Contains(t, keys, "in1")
	assert.Contains(t, keys, "out1")
	assert.Empty(t, resp.GetNextPageToken())
}

func TestArtifactServer_ListArtifactTasks_FilterByArtifactIds(t *testing.T) {
	s, _, _, _, _, art2 := seedArtifactTasks(t)
	resp, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{ArtifactIds: []string{art2.UUID}, PageSize: 50})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.GetTotalSize())
	assert.Equal(t, 2, len(resp.GetArtifactTasks()))
	// Ensure artifact_key values are as expected for art2 links
	keys := []string{resp.GetArtifactTasks()[0].GetArtifactKey(), resp.GetArtifactTasks()[1].GetArtifactKey()}
	assert.Contains(t, keys, "out1")
	assert.Contains(t, keys, "in2")
}

func TestArtifactServer_ListArtifactTasks_FilterByRunIds(t *testing.T) {
	s, _, _, t2, _, art2 := seedArtifactTasks(t)
	resp, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{RunIds: []string{serverRunID2}, PageSize: 50})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.GetTotalSize())
	assert.Equal(t, 1, len(resp.GetArtifactTasks()))
	at := resp.GetArtifactTasks()[0]
	assert.Equal(t, art2.UUID, at.GetArtifactId())
	assert.Equal(t, t2.UUID, at.GetTaskId())
	assert.Equal(t, "in2", at.GetArtifactKey())
}

func TestArtifactServer_ListArtifactTasks_ErrorWhenNoFilters(t *testing.T) {
	s, _, _, _, _, _ := seedArtifactTasks(t)
	_, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{PageSize: 2})
	assert.Error(t, err)
}

func TestArtifactServer_ListArtifactTasks_Pagination_TaskIds(t *testing.T) {
	s, _, t1, _, _, _ := seedArtifactTasks(t)
	page1, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{TaskIds: []string{t1.UUID}, PageSize: 1})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), page1.GetTotalSize())
	assert.Equal(t, 1, len(page1.GetArtifactTasks()))
	assert.NotEmpty(t, page1.GetNextPageToken())

	page2, err := s.ListArtifactTasks(ctxWithUser(), &apiv2beta1.ListArtifactTasksRequest{TaskIds: []string{t1.UUID}, PageToken: page1.GetNextPageToken(), PageSize: 1})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), page2.GetTotalSize())
	assert.Equal(t, 1, len(page2.GetArtifactTasks()))
	assert.Empty(t, page2.GetNextPageToken())

	id1 := page1.GetArtifactTasks()[0].GetId()
	id2 := page2.GetArtifactTasks()[0].GetId()
	assert.NotEqual(t, id1, id2)
}
