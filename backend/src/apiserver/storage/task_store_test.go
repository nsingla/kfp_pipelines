// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"fmt"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

const (
	testUUID1 = "123e4567-e89b-12d3-a456-426655441011"
	testUUID2 = "123e4567-e89b-12d3-a456-426655441012"
	testUUID3 = "123e4567-e89b-12d3-a456-426655441013"
	testUUID4 = "123e4567-e89b-12d3-a456-426655441014"
	testUUID5 = "123e4567-e89b-12d3-a456-426655441015"
)

// initializeTaskStore sets up a fake DB with a couple of runs and returns a TaskStore ready for testing.
func initializeTaskStore() (*DB, *TaskStore, *RunStore) {
	db := NewFakeDBOrFatal()
	fakeTime := util.NewFakeTimeForEpoch()
	// Seed a couple of runs to satisfy Task foreign key constraint.
	runStore := NewRunStore(db, fakeTime)
	run1 := &model.Run{
		UUID:         "run-1",
		ExperimentId: "exp-1",
		K8SName:      "run1",
		DisplayName:  "run1",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns1",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Running",
			State:            model.RuntimeStateRunning,
		},
	}
	run2 := &model.Run{
		UUID:         "run-2",
		ExperimentId: "exp-2",
		K8SName:      "run2",
		DisplayName:  "run2",
		StorageState: model.StorageStateAvailable,
		Namespace:    "ns2",
		RunDetails: model.RunDetails{
			CreatedAtInSec:   2,
			ScheduledAtInSec: 2,
			Conditions:       "Succeeded",
			State:            model.RuntimeStateSucceeded,
		},
	}
	_, _ = runStore.CreateRun(run1)
	_, _ = runStore.CreateRun(run2)

	// Create task store with controllable UUID generator
	taskStore := NewTaskStore(db, fakeTime, util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil))
	return db, taskStore, runStore
}

func createTaskPod(name, uid, typ string) *apiv2beta1.PipelineTaskDetail_TaskPod {
	return &apiv2beta1.PipelineTaskDetail_TaskPod{
		Name: name,
		Uid:  uid,
		Type: typ,
	}
}

func createTaskPodsAsJSONSlice(pods ...*apiv2beta1.PipelineTaskDetail_TaskPod) model.JSONSlice {
	podsAsSlice, err := model.ProtoSliceToJSONSlice(pods)
	if err != nil {
		panic(err)
	}
	return podsAsSlice
}

// Minimal test to ensure model<->DB mapping remains valid.
func TestTaskAPIFieldMap(t *testing.T) {
	for _, modelField := range (&model.Task{}).APIToModelFieldMap() {
		assert.Contains(t, taskColumns, modelField)
	}
}

func TestCreateTask_Success(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()
	pods := createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", "EXECUTOR"))
	task := &model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		Pods:             pods,
		Fingerprint:      "fp-1",
		Name:             "taskA",
		ParentTaskUUID:   "",
		Status:           1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        model.JSONData(map[string]interface{}{"k": "v"}),
	}

	created, err := taskStore.CreateTask(task)
	assert.NoError(t, err)
	assert.Equal(t, testUUID1, created.UUID)
	// CreatedAt and StartedInSec should be auto-populated to the same timestamp (fake time starts from 0 -> 1)
	assert.Equal(t, created.CreatedAtInSec, created.StartedInSec)
	assert.Greater(t, created.CreatedAtInSec, int64(0))

	// Verify it can be fetched back
	fetched, err := taskStore.GetTask(created.UUID)
	assert.NoError(t, err)
	assert.Equal(t, created.UUID, fetched.UUID)
	assert.Equal(t, "ns1", fetched.Namespace)
	assert.Equal(t, "pipeA", fetched.PipelineName)
	assert.Equal(t, "run-1", fetched.RunUUID)
	assert.Equal(t, pods, fetched.Pods)
	assert.Equal(t, "fp-1", fetched.Fingerprint)
	assert.Equal(t, "taskA", fetched.Name)
	assert.Equal(t, model.TaskStatus(1), fetched.Status)
	assert.Equal(t, model.TaskType(0), fetched.Type)
}

func TestGetTask_NotFound(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()
	_, err := taskStore.GetTask(testUUID1)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestListTasks_BasicAndFilters(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Create a parent task and two child tasks under different runs/pipelines
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	parent, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", "EXECUTOR")),
		Fingerprint:      "fp-parent",
		Status:           1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        model.JSONData{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID2, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		ParentTaskUUID:   parent.UUID,
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p2", "uid2", "EXECUTOR")),
		Fingerprint:      "fp-c1",
		Status:           1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        model.JSONData{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID3, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns2",
		PipelineName:     "pipeB",
		RunUUID:          "run-2",
		ParentTaskUUID:   parent.UUID,
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p3", "uid3", "EXECUTOR")),
		Fingerprint:      "fp-c2",
		Status:           1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        model.JSONData{},
	})
	assert.NoError(t, err)

	// List all tasks
	opts, _ := list.NewOptions(&model.Task{}, 10, "", nil)
	all, total, npt, err := taskStore.ListTasks(&model.FilterContext{}, opts)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(all))
	assert.Equal(t, 3, total)
	assert.Equal(t, "", npt)

	// Filter by RunUUID
	opts2, _ := list.NewOptions(&model.Task{}, 10, "", nil)
	runFiltered, total2, _, err := taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: "run-1"}}, opts2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(runFiltered))
	assert.Equal(t, 2, total2)

	// Filter by PipelineName
	opts3, _ := list.NewOptions(&model.Task{}, 10, "", nil)
	pipeFiltered, total3, _, err := taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.PipelineResourceType, ID: "pipeB"}}, opts3)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pipeFiltered))
	assert.Equal(t, 1, total3)

	// Filter by ParentTaskUUID (child tasks)
	opts4, _ := list.NewOptions(&model.Task{}, 10, "", nil)
	children, total4, _, err := taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: parent.UUID}}, opts4)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(children))
	assert.Equal(t, 2, total4)
}

func TestUpdateTask_Success(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	pod1 := createTaskPod("p1", "uid1", "EXECUTOR")
	pod2 := createTaskPod("p2", "uid2", "EXECUTOR")
	// Create a task
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	created, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		Pods:             createTaskPodsAsJSONSlice(pod1),
		Fingerprint:      "fp-0",
		Status:           1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Update some fields
	created.Name = "updatedName"
	created.Fingerprint = "fp-1"
	created.Pods = createTaskPodsAsJSONSlice(pod1, pod2)
	created.Status = 2
	updated, err := taskStore.UpdateTask(created)
	assert.NoError(t, err)
	assert.Equal(t, created.UUID, updated.UUID)
	assert.Equal(t, "updatedName", updated.Name)
	assert.Equal(t, "fp-1", updated.Fingerprint)
	assert.Equal(t, createTaskPodsAsJSONSlice(pod1, pod2), updated.Pods)
	assert.Equal(t, model.TaskStatus(2), updated.Status)
}

func TestGetChildTasks_ReturnsChildren(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	parent, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		Name:             "parent",
		DisplayName:      "Parent Task",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", "EXECUTOR")),
		Fingerprint:      "fp-p",
		Status:           1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID2, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		ParentTaskUUID:   parent.UUID,
		Name:             "child-a",
		DisplayName:      "First Child",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", "EXECUTOR")),
		Fingerprint:      "fp-a",
		Status:           1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID3, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		ParentTaskUUID:   parent.UUID,
		Name:             "child-b",
		DisplayName:      "Second Child",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", "EXECUTOR")),
		Fingerprint:      "fp-b",
		Status:           2,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	children, err := taskStore.GetChildTasks(parent.UUID)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(children))
}

func TestListTasks_FilterPredicates_EqualsOnColumns(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Seed 3 tasks across 2 runs with differing names, statuses and fingerprints.
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	_, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		Name:             "alpha",
		DisplayName:      "Alpha Task",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", "EXECUTOR")),
		Fingerprint:      "fp-alpha",
		Status:           1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID2, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		Name:             "beta",
		DisplayName:      "Beta Task",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", "EXECUTOR")),
		Fingerprint:      "fp-beta",
		Status:           1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID3, nil)
	_, err = taskStore.CreateTask(&model.Task{
		Namespace:        "ns2",
		PipelineName:     "pipeB",
		RunUUID:          "run-2",
		Name:             "gamma",
		DisplayName:      "Gamma Task",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", "EXECUTOR")),
		Fingerprint:      "fp-gamma",
		Status:           2,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// name == "beta"
	f1Proto := &apiv2beta1.Filter{
		Predicates: []*apiv2beta1.Predicate{
			{Key: "name", Operation: apiv2beta1.Predicate_EQUALS, Value: &apiv2beta1.Predicate_StringValue{StringValue: "beta"}},
		}}
	f1, err := filter.New(f1Proto)
	assert.NoError(t, err)
	opts1, err := list.NewOptions(&model.Task{}, 20, "", f1)
	assert.NoError(t, err)
	res1, total1, _, err := taskStore.ListTasks(&model.FilterContext{}, opts1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(res1))
	assert.Equal(t, 1, total1)
	assert.Equal(t, "beta", res1[0].Name)

	// status == 2
	f2Proto := &apiv2beta1.Filter{
		Predicates: []*apiv2beta1.Predicate{
			{Key: "status", Operation: apiv2beta1.Predicate_EQUALS, Value: &apiv2beta1.Predicate_IntValue{IntValue: 2}},
		},
	}
	f2, err := filter.New(f2Proto)
	assert.NoError(t, err)
	opts2, err := list.NewOptions(&model.Task{}, 20, "", f2)
	assert.NoError(t, err)
	res2, total2, _, err := taskStore.ListTasks(&model.FilterContext{}, opts2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(res2))
	assert.Equal(t, 1, total2)
	assert.Equal(t, model.TaskStatus(2), res2[0].Status)

	// cache_fingerprint == "fp-alpha"
	f3Proto := &apiv2beta1.Filter{
		Predicates: []*apiv2beta1.Predicate{
			{Key: "cache_fingerprint", Operation: apiv2beta1.Predicate_EQUALS, Value: &apiv2beta1.Predicate_StringValue{StringValue: "fp-alpha"}},
		}}
	f3, err := filter.New(f3Proto)
	assert.NoError(t, err)
	opts3, err := list.NewOptions(&model.Task{}, 20, "", f3)
	assert.NoError(t, err)
	res3, total3, _, err := taskStore.ListTasks(&model.FilterContext{}, opts3)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(res3))
	assert.Equal(t, 1, total3)
	assert.Equal(t, "fp-alpha", res3[0].Fingerprint)

	// Combined: run_id == "run-1" AND status == 1
	f4Proto := &apiv2beta1.Filter{
		Predicates: []*apiv2beta1.Predicate{
			{Key: "run_id", Operation: apiv2beta1.Predicate_EQUALS, Value: &apiv2beta1.Predicate_StringValue{StringValue: "run-1"}},
			{Key: "status", Operation: apiv2beta1.Predicate_EQUALS, Value: &apiv2beta1.Predicate_IntValue{IntValue: 1}},
		}}
	f4, err := filter.New(f4Proto)
	assert.NoError(t, err)
	opts4, err := list.NewOptions(&model.Task{}, 20, "", f4)
	assert.NoError(t, err)
	res4, total4, _, err := taskStore.ListTasks(&model.FilterContext{}, opts4)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(res4))
	assert.Equal(t, 2, total4)
}

func TestListTasks_PaginationWithToken(t *testing.T) {
	// Setup
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Seed 5 tasks. FakeTime.Now() increments per call, so CreatedAtInSec/StartedInSec are strictly increasing
	uuids := []string{testUUID1, testUUID2, testUUID3, testUUID4, testUUID5}
	for i, id := range uuids {
		// Control UUID so key tie-breaker is predictable if same timestamp ever occurred
		taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(id, nil)
		_, err := taskStore.CreateTask(&model.Task{
			Namespace:        "ns1",
			PipelineName:     "pipeA",
			RunUUID:          "run-1",
			Name:             fmt.Sprintf("task-%d", i+1),
			Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", "EXECUTOR")),
			Fingerprint:      fmt.Sprintf("fp-%d", i+1),
			Status:           1,
			StateHistory:     model.JSONSlice{},
			InputParameters:  model.JSONSlice{},
			OutputParameters: model.JSONSlice{},
			Type:             0,
			TypeAttrs:        map[string]interface{}{},
		})
		assert.NoError(t, err)
	}

	// Page 1
	opts1, _ := list.NewOptions(&model.Task{}, 2, "", nil)
	page1, total1, token1, err := taskStore.ListTasks(&model.FilterContext{}, opts1)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(page1))
	assert.Equal(t, 5, total1)
	assert.NotEmpty(t, token1)

	// Page 2 using token
	opts2, err := list.NewOptionsFromToken(token1, 2)
	assert.NoError(t, err)
	page2, total2, token2, err := taskStore.ListTasks(&model.FilterContext{}, opts2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(page2))
	assert.Equal(t, 5, total2)
	assert.NotEmpty(t, token2)

	// Page 3 using token (should be the last 1 item, then empty token)
	opts3, err := list.NewOptionsFromToken(token2, 2)
	assert.NoError(t, err)
	page3, total3, token3, err := taskStore.ListTasks(&model.FilterContext{}, opts3)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(page3))
	assert.Equal(t, 5, total3)
	assert.Empty(t, token3)
}

func TestTaskParameters_PersistAndFetch(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Build two simple Parameter protos for inputs and outputs
	inParam := &apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{
		Value:    "in-val",
		Name:     strPTR("in-name"),
		Producer: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOProducer{},
	}
	outParam := &apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{
		Value: "out-val",
		Name:  strPTR("out-name"),
		Producer: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOProducer{
			TaskName: "task-x",
			Key:      "param-y",
		},
	}
	inParams, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{inParam})
	assert.NoError(t, err)
	outParams, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{outParam})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	created, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", "EXECUTOR")),
		Fingerprint:      "fp-param",
		Status:           1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  inParams,
		OutputParameters: outParams,
		Type:             0,
		TypeAttrs:        model.JSONData{},
	})
	assert.NoError(t, err)

	fetched, err := taskStore.GetTask(created.UUID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(fetched.InputParameters))
	assert.Equal(t, 1, len(fetched.OutputParameters))
}

func TestHydrateArtifactsForTask_GetAndList(t *testing.T) {
	db, taskStore, _ := initializeTaskStore()
	defer db.Close()

	// Create a task under run-1
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
	task, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		PipelineName:     "pipeA",
		RunUUID:          "run-1",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", "EXECUTOR")),
		Fingerprint:      "fp-art",
		Status:           1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        model.JSONData{},
	})
	assert.NoError(t, err)

	// Create two artifacts via the ArtifactStore
	artifactStore := NewArtifactStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(testUUID2, nil))
	artIn, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      0,
		Uri:       strPTR("s3://bucket/in"),
		Name:      "in-art",
	})
	assert.NoError(t, err)
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID3, nil)
	artOut, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      0,
		Uri:       strPTR("s3://bucket/out"),
		Name:      "out-art",
	})
	assert.NoError(t, err)

	// Link artifacts to task via artifact_tasks
	ats1 := NewArtifactTaskStore(db, util.NewFakeUUIDGeneratorOrFatal("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1", nil))
	// Input link with no producer fields -> ResolvedValue
	_, err = ats1.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:       artIn.UUID,
		TaskID:           task.UUID,
		Type:             apiv2beta1.ArtifactTaskType_INPUT,
		RunUUID:          task.RunUUID,
		ProducerTaskName: "",
		ProducerKey:      "",
	})
	assert.NoError(t, err)
	// Output link with producer fields -> PipelineChannel
	ats2 := NewArtifactTaskStore(db, util.NewFakeUUIDGeneratorOrFatal("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2", nil))
	_, err = ats2.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:       artOut.UUID,
		TaskID:           task.UUID,
		Type:             apiv2beta1.ArtifactTaskType_OUTPUT,
		RunUUID:          task.RunUUID,
		ProducerTaskName: "producer-task",
		ProducerKey:      "output-key",
	})
	assert.NoError(t, err)

	// Verify GetTask hydrates artifacts
	fetched, err := taskStore.GetTask(task.UUID)
	assert.NoError(t, err)
	if assert.Equal(t, 1, len(fetched.InputArtifactsHydrated)) {
		ia := fetched.InputArtifactsHydrated[0]
		assert.Equal(t, ia.Producer, model.IOProducer{
			TaskName: "",
			Key:      "",
		})
		if assert.NotNil(t, ia.Value) {
			assert.Equal(t, "in-art", ia.Value.Name)
		}
	}
	if assert.Equal(t, 1, len(fetched.OutputArtifactsHydrated)) {
		oa := fetched.OutputArtifactsHydrated[0]
		assert.Equal(t, "producer-task", oa.Producer.TaskName)
		assert.Equal(t, "output-key", oa.Producer.Key)
		if assert.NotNil(t, oa.Value) {
			assert.Equal(t, "out-art", oa.Value.Name)
		}
	}

	// Verify ListTasks hydrates artifacts as well
	opts, _ := list.NewOptions(&model.Task{}, 10, "", nil)
	tasks, _, _, err := taskStore.ListTasks(&model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: task.RunUUID}}, opts)
	assert.NoError(t, err)
	var found *model.Task
	for _, tsk := range tasks {
		if tsk.UUID == task.UUID {
			found = tsk
			break
		}
	}
	if assert.NotNil(t, found) {
		assert.Equal(t, 1, len(found.InputArtifactsHydrated))
		assert.Equal(t, 1, len(found.OutputArtifactsHydrated))
	}
}
