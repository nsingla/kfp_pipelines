package driver

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const testPipelineName = "test-pipeline"
const testNamespace = "test-namespace"

// LoadPipelineSpecFromYAML loads a pipeline spec from a YAML file path
func LoadPipelineSpecFromYAML(path string) (*pipelinespec.PipelineSpec, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	// Convert YAML -> JSON, then use protojson to honor proto field names
	jsonBytes, err := yaml.YAMLToJSON(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	var spec pipelinespec.PipelineSpec
	um := protojson.UnmarshalOptions{
		DiscardUnknown: true, // tolerate extra fields
	}
	if err := um.Unmarshal(jsonBytes, &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PipelineSpec (protojson): %w", err)
	}
	return &spec, nil
}

func basicRuntimeConfig() *pipelinespec.PipelineJob_RuntimeConfig {
	return &pipelinespec.PipelineJob_RuntimeConfig{
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
}

// MockDriverAPI provides a mock implementation of DriverAPI for testing
type MockDriverAPI struct {
	runs          map[string]*apiv2beta1.Run
	tasks         map[string]*apiv2beta1.PipelineTaskDetail
	artifacts     map[string]*apiv2beta1.Artifact
	artifactTasks map[string]*apiv2beta1.ArtifactTask
}

// NewMockDriverAPI creates a new mock driver API
func NewMockDriverAPI() *MockDriverAPI {
	return &MockDriverAPI{
		runs:          make(map[string]*apiv2beta1.Run),
		tasks:         make(map[string]*apiv2beta1.PipelineTaskDetail),
		artifacts:     make(map[string]*apiv2beta1.Artifact),
		artifactTasks: make(map[string]*apiv2beta1.ArtifactTask),
	}
}

func (m *MockDriverAPI) GetRun(ctx context.Context, req *apiv2beta1.GetRunRequest) (*apiv2beta1.Run, error) {
	if run, exists := m.runs[req.RunId]; exists {
		// Create a copy of the run to populate with tasks
		populatedRun := &apiv2beta1.Run{
			RunId:          run.RunId,
			DisplayName:    run.DisplayName,
			PipelineSource: &apiv2beta1.Run_PipelineSpec{PipelineSpec: run.GetPipelineSpec()},
			RuntimeConfig:  run.RuntimeConfig,
			State:          run.State,
			Tasks:          []*apiv2beta1.PipelineTaskDetail{},
		}

		// Find all tasks for this run
		for _, task := range m.tasks {
			if task.RunId == req.RunId {
				// Create a copy of the task to populate with artifacts
				populatedTask := m.hydrateTask(task)
				populatedRun.Tasks = append(populatedRun.Tasks, populatedTask)
			}
		}

		return populatedRun, nil
	}
	return nil, fmt.Errorf("run not found: %s", req.RunId)
}

func (m *MockDriverAPI) hydrateTask(task *apiv2beta1.PipelineTaskDetail) *apiv2beta1.PipelineTaskDetail {
	// Create a copy of the task to populate with artifacts
	populatedTask := &apiv2beta1.PipelineTaskDetail{
		TaskId:         task.TaskId,
		Name:           task.Name,
		DisplayName:    task.DisplayName,
		RunId:          task.RunId,
		Type:           task.Type,
		Status:         task.Status,
		ParentTaskId:   task.ParentTaskId,
		Pods:           task.Pods,
		StatusMetadata: task.StatusMetadata,
		Inputs:         &apiv2beta1.PipelineTaskDetail_InputOutputs{},
		Outputs:        &apiv2beta1.PipelineTaskDetail_InputOutputs{},
	}

	// Copy existing parameters if they exist
	if task.Inputs != nil {
		populatedTask.Inputs.Parameters = task.Inputs.Parameters
	}
	if task.Outputs != nil {
		populatedTask.Outputs.Parameters = task.Outputs.Parameters
	}

	// Find artifacts associated with this task
	var inputArtifacts []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact
	var outputArtifacts []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact

	for _, artifactTask := range m.artifactTasks {
		if artifactTask.TaskId == task.TaskId {
			// Get the associated artifact
			if artifact, exists := m.artifacts[artifactTask.ArtifactId]; exists {
				ioArtifact := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
					Artifacts: []*apiv2beta1.Artifact{artifact},
					Type:      artifactTask.Type,
				}

				// Set the source based on producer information
				if artifactTask.ProducerTaskName != "" && artifactTask.ProducerKey != "" {
					ioArtifact.Source = &apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact_Producer{
						Producer: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOProducer{
							TaskName: artifactTask.ProducerTaskName,
							Key:      artifactTask.ProducerKey,
						},
					}
				} else {
					// Use parameter name if no producer specified
					ioArtifact.Source = &apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact_ParameterName{
						ParameterName: artifact.Name,
					}
				}

				// Determine if this is an input or output artifact based on ArtifactTaskType
				switch artifactTask.Type {
				case apiv2beta1.IOType_INPUT, apiv2beta1.IOType_ITERATOR_INPUT:
					inputArtifacts = append(inputArtifacts, ioArtifact)
				case apiv2beta1.IOType_OUTPUT:
					outputArtifacts = append(outputArtifacts, ioArtifact)
				}
			}
		}
	}

	// Set the artifacts on the task
	populatedTask.Inputs.Artifacts = inputArtifacts
	populatedTask.Outputs.Artifacts = outputArtifacts

	return populatedTask
}

func (m *MockDriverAPI) CreateTask(ctx context.Context, req *apiv2beta1.CreateTaskRequest) (*apiv2beta1.PipelineTaskDetail, error) {
	task := req.Task
	if task.TaskId == "" {
		uuid, _ := uuid.NewRandom()
		task.TaskId = uuid.String()
	}
	m.tasks[task.TaskId] = task
	return task, nil
}

func (m *MockDriverAPI) UpdateTask(ctx context.Context, req *apiv2beta1.UpdateTaskRequest) (*apiv2beta1.PipelineTaskDetail, error) {
	if _, exists := m.tasks[req.TaskId]; !exists {
		return nil, fmt.Errorf("task not found: %s", req.TaskId)
	}
	task := req.Task
	task.TaskId = req.TaskId
	m.tasks[req.TaskId] = task
	task = m.hydrateTask(task)
	return task, nil
}

func (m *MockDriverAPI) GetTask(ctx context.Context, req *apiv2beta1.GetTaskRequest) (*apiv2beta1.PipelineTaskDetail, error) {
	if task, exists := m.tasks[req.TaskId]; exists {
		task = m.hydrateTask(m.tasks[req.TaskId])
		return task, nil
	}

	return nil, fmt.Errorf("task not found: %s", req.TaskId)
}

func (m *MockDriverAPI) ListTasks(ctx context.Context, req *apiv2beta1.ListTasksRequest) (*apiv2beta1.ListTasksResponse, error) {
	var tasks []*apiv2beta1.PipelineTaskDetail

	// Filter by run ID if specified
	if runId := req.GetRunId(); runId != "" {
		for _, task := range m.tasks {
			if task.RunId == runId {
				tasks = append(tasks, task)
			}
		}
	} else if parentId := req.GetParentId(); parentId != "" {
		// Filter by parent task ID
		for _, task := range m.tasks {
			if task.ParentTaskId != nil && *task.ParentTaskId == parentId {
				tasks = append(tasks, task)
			}
		}
	} else {
		// Return all tasks
		for _, task := range m.tasks {
			tasks = append(tasks, task)
		}
	}

	var hydratedTasks []*apiv2beta1.PipelineTaskDetail
	for _, task := range tasks {
		hydratedTasks = append(hydratedTasks, m.hydrateTask(task))
	}

	return &apiv2beta1.ListTasksResponse{
		Tasks:     hydratedTasks,
		TotalSize: int32(len(tasks)),
	}, nil
}

func (m *MockDriverAPI) CreateArtifact(ctx context.Context, req *apiv2beta1.CreateArtifactRequest) (*apiv2beta1.Artifact, error) {
	artifact := req.Artifact
	if artifact.ArtifactId == "" {
		uuid, _ := uuid.NewRandom()
		artifact.ArtifactId = uuid.String()
	}
	m.artifacts[artifact.ArtifactId] = artifact
	return artifact, nil
}

func (m *MockDriverAPI) ListArtifactTasks(ctx context.Context, req *apiv2beta1.ListArtifactTasksRequest) (*apiv2beta1.ListArtifactTasksResponse, error) {
	var artifactTasks []*apiv2beta1.ArtifactTask
	for _, at := range m.artifactTasks {
		artifactTasks = append(artifactTasks, at)
	}
	return &apiv2beta1.ListArtifactTasksResponse{
		ArtifactTasks: artifactTasks,
		TotalSize:     int32(len(artifactTasks)),
	}, nil
}

func (m *MockDriverAPI) CreateArtifactTask(ctx context.Context, req *apiv2beta1.CreateArtifactTaskRequest) (*apiv2beta1.ArtifactTask, error) {
	artifactTask := req.ArtifactTask
	if artifactTask.Id == "" {
		uuid, _ := uuid.NewRandom()
		artifactTask.Id = uuid.String()
	}
	m.artifactTasks[artifactTask.Id] = artifactTask
	return artifactTask, nil
}

func (m *MockDriverAPI) CreateArtifactTasks(ctx context.Context, req *apiv2beta1.CreateArtifactTasksBulkRequest) (*apiv2beta1.CreateArtifactTasksBulkResponse, error) {
	var createdTasks []*apiv2beta1.ArtifactTask
	for _, at := range req.ArtifactTasks {
		if at.Id == "" {
			uuid, _ := uuid.NewRandom()
			at.Id = uuid.String()
		}
		m.artifactTasks[at.Id] = at
		createdTasks = append(createdTasks, at)
	}
	return &apiv2beta1.CreateArtifactTasksBulkResponse{
		ArtifactTasks: createdTasks,
	}, nil
}

// AddRun adds a run to the mock for testing
func (m *MockDriverAPI) AddRun(run *apiv2beta1.Run) {
	if run.RunId == "" {
		uuid, _ := uuid.NewRandom()
		run.RunId = uuid.String()
	}
	m.runs[run.RunId] = run
}

// TestSetup provides a complete test setup with mock driver API
type TestSetup struct {
	DriverAPI *MockDriverAPI
}

// NewTestSetup creates a new test setup
func NewTestSetup(t *testing.T) *TestSetup {
	t.Helper()
	proxy.InitializeConfigWithEmptyForTests()
	return &TestSetup{
		DriverAPI: NewMockDriverAPI(),
	}
}

// CreateTestRun creates a test run with basic configuration
func (ts *TestSetup) CreateTestRun(t *testing.T, pipelineName string) *apiv2beta1.Run {
	t.Helper()

	pipelineSpec := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"pipelineInfo": {
				Kind: &structpb.Value_StructValue{
					StructValue: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"name": {
								Kind: &structpb.Value_StringValue{StringValue: pipelineName},
							},
						},
					},
				},
			},
			"root": {
				Kind: &structpb.Value_StructValue{
					StructValue: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"dag": {
								Kind: &structpb.Value_StructValue{
									StructValue: &structpb.Struct{
										Fields: map[string]*structpb.Value{
											"tasks": {
												Kind: &structpb.Value_StructValue{
													StructValue: &structpb.Struct{Fields: map[string]*structpb.Value{}},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	uuid, _ := uuid.NewRandom()
	run := &apiv2beta1.Run{
		RunId:          uuid.String(),
		DisplayName:    fmt.Sprintf("test-run-%s-%d", pipelineName, time.Now().Unix()),
		PipelineSource: &apiv2beta1.Run_PipelineSpec{PipelineSpec: pipelineSpec},
		RuntimeConfig:  &apiv2beta1.RuntimeConfig{},
		State:          apiv2beta1.RuntimeState_RUNNING,
	}

	ts.DriverAPI.AddRun(run)
	return run
}

// CreateTestTask creates a test task with the given configuration
func (ts *TestSetup) CreateTestTask(
	t *testing.T,
	runID,
	taskName string,
	taskType apiv2beta1.PipelineTaskDetail_TaskType,
	inputParams, outputParams []*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter,
) *apiv2beta1.PipelineTaskDetail {
	t.Helper()

	podUuid, _ := uuid.NewRandom()
	task := &apiv2beta1.PipelineTaskDetail{
		Name:        taskName,
		DisplayName: taskName,
		RunId:       runID,
		Type:        taskType,
		Status:      apiv2beta1.PipelineTaskDetail_RUNNING,
		Pods: []*apiv2beta1.PipelineTaskDetail_TaskPod{
			{
				Name: fmt.Sprintf("%s-pod", taskName),
				Uid:  podUuid.String(),
				Type: apiv2beta1.PipelineTaskDetail_DRIVER,
			},
		},
		Inputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{
			Parameters: inputParams,
		},
		Outputs: &apiv2beta1.PipelineTaskDetail_InputOutputs{
			Parameters: outputParams,
		},
	}

	createdTask, err := ts.DriverAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{
		Task: task,
	})
	require.NoError(t, err)
	return createdTask
}

// CreateTestArtifact creates a test artifact with the given configuration
func (ts *TestSetup) CreateTestArtifact(t *testing.T, name, artifactType string) *apiv2beta1.Artifact {
	t.Helper()

	artifact := &apiv2beta1.Artifact{
		Name: name,
		Type: apiv2beta1.Artifact_Dataset, // Default type
	}

	// Set specific type if provided
	switch artifactType {
	case "model":
		artifact.Type = apiv2beta1.Artifact_Model
	case "metric":
		artifact.Type = apiv2beta1.Artifact_Metric
	}

	createdArtifact, err := ts.DriverAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact: artifact,
	})
	require.NoError(t, err)
	return createdArtifact
}

// CreateTestArtifactTask creates an artifact-task relationship
func (ts *TestSetup) CreateTestArtifactTask(t *testing.T, artifactID, taskID, runID, producerTaskName, producerKey string, artifactType apiv2beta1.IOType) *apiv2beta1.ArtifactTask {
	t.Helper()

	artifactTask := &apiv2beta1.ArtifactTask{
		ArtifactId:       artifactID,
		TaskId:           taskID,
		RunId:            runID,
		Type:             artifactType,
		ProducerTaskName: producerTaskName,
		ProducerKey:      producerKey,
	}

	createdArtifactTask, err := ts.DriverAPI.CreateArtifactTask(context.Background(), &apiv2beta1.CreateArtifactTaskRequest{
		ArtifactTask: artifactTask,
	})
	require.NoError(t, err)
	return createdArtifactTask
}

func CreateTestOptions(t *testing.T, driverAPI common.DriverAPI, run *apiv2beta1.Run) *common.Options {
	t.Helper()

	// Create a basic component spec
	component := &pipelinespec.ComponentSpec{
		Implementation: &pipelinespec.ComponentSpec_Dag{
			Dag: &pipelinespec.DagSpec{
				Tasks: map[string]*pipelinespec.PipelineTaskSpec{},
			},
		},
	}

	// Create a parent task
	parentUuid, _ := util.NewFakeUUIDGeneratorOrFatal("parent-task", nil).NewRandom()
	parentTask := &apiv2beta1.PipelineTaskDetail{
		TaskId:      parentUuid.String(),
		Name:        "parent-task",
		DisplayName: "Parent Task",
		RunId:       run.RunId,
		Type:        apiv2beta1.PipelineTaskDetail_DAG,
		Status:      apiv2beta1.PipelineTaskDetail_RUNNING,
	}

	return &common.Options{
		PipelineName:   "test-pipeline",
		Run:            run,
		Component:      component,
		ParentTask:     parentTask,
		DriverAPI:      driverAPI,
		IterationIndex: -1, // Not an iteration
	}
}

// CreateTestWorkflow creates a test Argo workflow
func CreateTestWorkflow(name, namespace string) *util.Workflow {
	return util.NewWorkflow(&v1alpha1.Workflow{
		TypeMeta:   v1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: v1alpha1.WorkflowSpec{
			Entrypoint: "main",
			Templates: []v1alpha1.Template{
				{
					Name: "main",
					Container: &corev1.Container{
						Image:   "alpine:latest",
						Command: []string{"echo"},
						Args:    []string{"hello world"},
					},
				},
			},
		},
		Status: v1alpha1.WorkflowStatus{
			Phase: v1alpha1.WorkflowRunning,
		},
	})
}

// AssertTaskStatus is a helper to check task status
func AssertTaskStatus(t *testing.T, driverAPI common.DriverAPI, taskID string, expectedStatus apiv2beta1.PipelineTaskDetail_TaskState) {
	t.Helper()

	ctx := context.Background()
	task, err := driverAPI.GetTask(ctx, &apiv2beta1.GetTaskRequest{TaskId: taskID})
	require.NoError(t, err)
	assert.Equal(t, expectedStatus, task.Status)
}

// AssertTaskType is a helper to check task type
func AssertTaskType(t *testing.T, driverAPI common.DriverAPI, taskID string, expectedType apiv2beta1.PipelineTaskDetail_TaskType) {
	t.Helper()

	ctx := context.Background()
	task, err := driverAPI.GetTask(ctx, &apiv2beta1.GetTaskRequest{TaskId: taskID})
	require.NoError(t, err)
	assert.Equal(t, expectedType, task.Type)
}

// CreateParameter creates a test parameter with the given name and value
func CreateParameter(name, value string, producer *apiv2beta1.PipelineTaskDetail_InputOutputs_IOProducer) *apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter {
	if name != "" && producer != nil {
		panic("Cannot specify both name and producer")
	}
	val, _ := structpb.NewValue(value)
	param := &apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{
		Value: val,
	}
	if name != "" {
		param.Source = &apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter_ParameterName{
			ParameterName: name,
		}
	} else {
		param.Source = &apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter_Producer{
			Producer: producer,
		}
	}

	return param
}

// CreateParameterWithProducer creates a parameter with a producer reference
func CreateParameterWithProducer(value, taskName, key string) *apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter {
	val, _ := structpb.NewValue(value)
	return &apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{
		Value: val,
		Source: &apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter_Producer{
			Producer: &apiv2beta1.PipelineTaskDetail_InputOutputs_IOProducer{
				TaskName: taskName,
				Key:      key,
			},
		},
	}
}

// CreateStatusMetadata creates test status metadata
func CreateStatusMetadata(message string, properties map[string]interface{}) *apiv2beta1.PipelineTaskDetail_StatusMetadata {
	customProps := make(map[string]*structpb.Value)
	for k, v := range properties {
		val, _ := structpb.NewValue(v)
		customProps[k] = val
	}

	return &apiv2beta1.PipelineTaskDetail_StatusMetadata{
		Message:          message,
		CustomProperties: customProps,
	}
}

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
	task1 := testSetup.CreateTestTask(t, run.RunId, "producer-task", apiv2beta1.PipelineTaskDetail_RUNTIME,
		[]*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{
			CreateParameter("input1", "test-input1", nil),
			CreateParameter("", "test-input2", &apiv2beta1.PipelineTaskDetail_InputOutputs_IOProducer{TaskName: "some-task", Key: "some-key"}),
		},
		[]*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{
			CreateParameter("output1", "test-output1", nil),
			CreateParameter("output2", "test-output2", nil),
		})
	task2 := testSetup.CreateTestTask(t, run.RunId, "consumer-task", apiv2beta1.PipelineTaskDetail_RUNTIME,
		[]*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{
			CreateParameterWithProducer("test-input3", "producer-task", "output1"),
			CreateParameter("input4", "test-input4", nil),
		},
		[]*apiv2beta1.PipelineTaskDetail_InputOutputs_Parameter{
			CreateParameter("output3", "test-output3", nil),
		})

	// Create test artifacts
	artifact1 := testSetup.CreateTestArtifact(t, "output-data", "dataset")
	artifact2 := testSetup.CreateTestArtifact(t, "trained-model", "model")

	// Create artifact-task relationships
	// task1 produces artifact1 (output)
	testSetup.CreateTestArtifactTask(t, artifact1.ArtifactId, task1.TaskId, run.RunId, task1.Name, "output", apiv2beta1.IOType_OUTPUT)
	// task2 consumes artifact1 (input)
	testSetup.CreateTestArtifactTask(t, artifact1.ArtifactId, task2.TaskId, run.RunId, task1.Name, "output", apiv2beta1.IOType_INPUT)
	// task2 produces artifact2 (output)
	testSetup.CreateTestArtifactTask(t, artifact2.ArtifactId, task2.TaskId, run.RunId, task2.Name, "model", apiv2beta1.IOType_OUTPUT)

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
	assert.Equal(t, "output", inputArtifact.GetProducer().Key)
}
