# Artifact Store Integration Test Plan for MLMD Removal

## Executive Summary

This document outlines a comprehensive testing plan for validating the MLMD removal implementation, focusing on the new Artifact Store API, namespace isolation, and end-to-end artifact handling. The plan follows existing test patterns in `backend/test/v2/api` and enhances `backend/test/end2end` validations.

## Test Architecture Overview

### Test Structure Pattern (Following `/backend/test/v2/api`)

```
backend/test/v2/api/
├── artifact_store_api_test.go        # New: Artifact API tests
├── artifact_task_api_test.go         # New: Artifact-Task relationship tests
├── integration_suite_test.go         # Enhanced: Add artifact clients
├── test_context.go                   # Enhanced: Add artifact context
└── matcher/
    ├── artifact_matcher.go           # New: Artifact validation matchers
    └── task_matcher.go               # New: Task validation matchers
```

### End-to-End Enhancements

```
backend/test/end2end/
├── pipeline_e2e_test.go              # Enhanced: Task and artifact validation
└── utils/
    └── e2e_utils.go                  # Enhanced: Artifact URI validation
```

## Phase 1: Integration Test Infrastructure

### 1.1 Test Context Enhancement

**File**: `backend/test/v2/api/test_context.go`

```go
type TestContext struct {
    // Existing fields
    TestStartTimeUTC time.Time
    Pipeline         Pipeline
    PipelineRun      PipelineRun
    Experiment       Experiment

    // New: Artifact testing context
    Artifacts        ArtifactContext
    Tasks           TaskContext
}

type ArtifactContext struct {
    CreatedArtifactIds     []string
    CreatedArtifactTaskIds []string
    ExpectedArtifacts      []*v2beta1.Artifact
}

type TaskContext struct {
    CreatedTaskIds  []string
    ExpectedTasks   []*v2beta1.PipelineTaskDetail
}
```

### 1.2 Client Integration

**File**: `backend/test/v2/api/integration_suite_test.go`

Add artifact and task service clients to the global test infrastructure:

```go
var (
    // Existing clients
    pipelineUploadClient apiserver.PipelineUploadInterface
    pipelineClient       *apiserver.PipelineClient
    runClient            *apiserver.RunClient
    experimentClient     *apiserver.ExperimentClient
    recurringRunClient   *apiserver.RecurringRunClient

    // New: Artifact service clients
    artifactClient       *apiserver.ArtifactClient
    taskClient          *apiserver.TaskClient
    k8Client            *kubernetes.Clientset
)
```

## Phase 2: Core Artifact Store API Tests

### 2.1 Artifact CRUD Operations Test

**File**: `backend/test/v2/api/artifact_store_api_test.go`

#### 2.1.1 Positive Tests

```go
var _ = Describe("Artifact Store API Tests >", Label(constants.POSITIVE, "Artifact", constants.APIServerTests, constants.FullRegression), func() {

    Context("Basic Artifact Operations >", func() {
        It("Create a basic artifact with minimal fields", func() {
            // Test CreateArtifact with required fields only
        })

        It("Create a metric artifact with number value", func() {
            // Test artifact with type=Metric and NumberValue
        })

        It("Create a classification metric artifact with metadata", func() {
            // Test complex artifacts with JSON metadata
        })

        It("Get artifact by ID", func() {
            // Test GetArtifact retrieval and field validation
        })

        It("List artifacts by namespace", func() {
            // Test ListArtifacts with namespace filtering
        })

        It("List artifacts with pagination", func() {
            // Test pagination parameters and next_page_token
        })
    })

    Context("Bulk Operations >", func() {
        It("Create multiple artifacts in bulk", func() {
            // Test CreateArtifactsBulk with various artifact types
        })

        It("Create bulk artifacts with task associations", func() {
            // Test bulk creation with artifact-task relationships
        })

        It("Handle bulk operation partial failures", func() {
            // Test transaction rollback on failures
        })
    })

    Context("Artifact Filtering and Sorting >", func() {
        It("Filter artifacts by type", func() {
            // Test filtering by ArtifactType enum values
        })

        It("Filter artifacts by URI pattern", func() {
            // Test URI-based filtering for object store artifacts
        })

        It("Sort artifacts by creation date", func() {
            // Test sorting by CreatedAtInSec field
        })

        It("Sort artifacts by name", func() {
            // Test alphabetical sorting by Name field
        })
    })
})
```

#### 2.1.2 Namespace Isolation Tests

```go
Context("Namespace Isolation >", func() {
    It("User can only access artifacts in authorized namespaces", func() {
        // Create artifacts in multiple namespaces
        // Verify RBAC enforcement prevents cross-namespace access
    })

    It("List artifacts respects namespace boundaries", func() {
        // Test that ListArtifacts only returns permitted artifacts
    })

    It("Artifact creation enforces namespace ownership", func() {
        // Verify users cannot create artifacts in unauthorized namespaces
    })

    It("Multi-user mode prevents artifact leakage", func() {
        // Test complete isolation in multi-tenant environment
    })
})
```

#### 2.1.3 Security and Authorization Tests

```go
Context("Security and Authorization >", func() {
    It("Rejects artifacts without proper RBAC permissions", func() {
        // Test authorization failure scenarios
    })

    It("Validates artifact namespace matches task namespace", func() {
        // Test cross-namespace artifact-task association prevention
    })

    It("Handles invalid artifact metadata securely", func() {
        // Test JSON injection prevention and validation
    })
})
```

### 2.2 Artifact-Task Relationship Tests

**File**: `backend/test/v2/api/artifact_task_api_test.go`

```go
var _ = Describe("Artifact-Task Relationship API Tests >", func() {

    Context("Artifact-Task Associations >", func() {
        It("Create INPUT artifact-task relationship", func() {
            // Test IOType_INPUT relationships
        })

        It("Create OUTPUT artifact-task relationship", func() {
            // Test IOType_OUTPUT relationships
        })

        It("Create bulk artifact-task relationships", func() {
            // Test CreateArtifactTasksBulk operation
        })
    })

    Context("Artifact-Task Filtering >", func() {
        It("List artifact-tasks by task ID", func() {
            // Test filtering by TaskIds parameter
        })

        It("List artifact-tasks by run ID", func() {
            // Test filtering by RunIds parameter
        })

        It("List artifact-tasks by artifact ID", func() {
            // Test filtering by ArtifactIds parameter
        })

        It("List artifact-tasks by IOType", func() {
            // Test filtering by Type parameter (INPUT/OUTPUT)
        })

        It("List artifact-tasks with multiple filters", func() {
            // Test combined filtering scenarios
        })
    })

    Context("Producer Information >", func() {
        It("Stores correct task producer information", func() {
            // Validate Producer field contains task name and iteration
        })

        It("Handles producer iteration index correctly", func() {
            // Test iteration-specific producer data
        })
    })
})
```

### 2.3 Performance and Edge Case Tests

```go
Context("Performance and Edge Cases >", func() {
    It("Handles large artifact metadata efficiently", func() {
        // Test with complex JSON metadata (within limits)
    })

    It("Processes high-volume artifact creation", func() {
        // Test bulk operations with 100+ artifacts
    })

    It("Manages concurrent artifact access correctly", func() {
        // Test concurrent read/write operations
    })

    It("Handles database transaction failures gracefully", func() {
        // Test transaction rollback scenarios
    })
})
```

## Phase 3: Artifact Validation Matchers

### 3.1 Artifact Matchers

**File**: `backend/test/v2/api/matcher/artifact_matcher.go`

```go
package matcher

// MatchArtifacts - Deep compare artifacts including metadata validation
func MatchArtifacts(actual *v2beta1.Artifact, expected *v2beta1.Artifact) {
    ginkgo.GinkgoHelper()

    // Basic field validation
    gomega.Expect(actual.ArtifactId).To(gomega.Not(gomega.BeEmpty()), "Artifact ID is empty")
    gomega.Expect(actual.Name).To(gomega.Equal(expected.Name), "Artifact name not matching")
    gomega.Expect(actual.Type).To(gomega.Equal(expected.Type), "Artifact type not matching")
    gomega.Expect(actual.Namespace).To(gomega.Equal(expected.Namespace), "Artifact namespace not matching")

    // Time validation
    actualTime := time.Unix(actual.CreatedAt.Seconds, 0).UTC()
    expectedTime := time.Unix(expected.CreatedAt.Seconds, 0).UTC()
    gomega.Expect(actualTime.After(expectedTime) || actualTime.Equal(expectedTime)).To(gomega.BeTrue(),
        "Artifact creation time validation failed")

    // URI validation for artifacts with storage
    if expected.Uri != "" {
        gomega.Expect(actual.Uri).To(gomega.Not(gomega.BeEmpty()), "Artifact URI should not be empty")
        gomega.Expect(actual.Uri).To(gomega.MatchRegexp(`^(gs://|s3://|minio://)`),
            "Artifact URI should follow object storage pattern")
    }

    // Metadata validation for complex artifacts
    if expected.Metadata != nil {
        ValidateArtifactMetadata(actual.Metadata, expected.Metadata)
    }

    // Number value validation for metric artifacts
    if expected.Type == v2beta1.Artifact_Metric {
        gomega.Expect(actual.NumberValue).To(gomega.Not(gomega.BeNil()),
            "Metric artifacts must have NumberValue")
        if expected.NumberValue != nil {
            gomega.Expect(*actual.NumberValue).To(gomega.BeNumerically("~", *expected.NumberValue, 0.001),
                "Metric value not matching")
        }
    }
}

// ValidateArtifactMetadata - Validates JSON metadata fields
func ValidateArtifactMetadata(actual *structpb.Struct, expected *structpb.Struct) {
    ginkgo.GinkgoHelper()

    if expected == nil && actual == nil {
        return
    }

    gomega.Expect(actual).To(gomega.Not(gomega.BeNil()), "Actual metadata should not be nil")
    gomega.Expect(expected).To(gomega.Not(gomega.BeNil()), "Expected metadata should not be nil")

    // Convert to maps for deep comparison
    actualMap := actual.AsMap()
    expectedMap := expected.AsMap()

    for key, expectedValue := range expectedMap {
        actualValue, exists := actualMap[key]
        gomega.Expect(exists).To(gomega.BeTrue(), fmt.Sprintf("Metadata key '%s' missing", key))
        gomega.Expect(actualValue).To(gomega.Equal(expectedValue),
            fmt.Sprintf("Metadata value for key '%s' not matching", key))
    }
}
```

### 3.2 Task Matchers

**File**: `backend/test/v2/api/matcher/task_matcher.go`

```go
// MatchTasks - Deep compare task details including artifact associations
func MatchTasks(actual *v2beta1.PipelineTaskDetail, expected *v2beta1.PipelineTaskDetail) {
    ginkgo.GinkgoHelper()

    // Basic task validation
    gomega.Expect(actual.TaskId).To(gomega.Not(gomega.BeEmpty()), "Task ID is empty")
    gomega.Expect(actual.TaskName).To(gomega.Equal(expected.TaskName), "Task name not matching")
    gomega.Expect(actual.RunId).To(gomega.Equal(expected.RunId), "Run ID not matching")

    // State validation
    gomega.Expect(actual.State).To(gomega.Not(gomega.BeNil()), "Task state should not be nil")

    // Artifact validation
    if len(expected.Inputs.Artifacts) > 0 || len(expected.Outputs.Artifacts) > 0 {
        ValidateTaskArtifacts(actual, expected)
    }
}

// ValidateTaskArtifacts - Validates artifact inputs and outputs for a task
func ValidateTaskArtifacts(actual *v2beta1.PipelineTaskDetail, expected *v2beta1.PipelineTaskDetail) {
    ginkgo.GinkgoHelper()

    // Input artifacts validation
    gomega.Expect(len(actual.Inputs.Artifacts)).To(gomega.Equal(len(expected.Inputs.Artifacts)),
        "Input artifact count not matching")

    for key, expectedArtifact := range expected.Inputs.Artifacts {
        actualArtifact, exists := actual.Inputs.Artifacts[key]
        gomega.Expect(exists).To(gomega.BeTrue(), fmt.Sprintf("Input artifact '%s' missing", key))
        MatchArtifacts(actualArtifact, expectedArtifact)
    }

    // Output artifacts validation
    gomega.Expect(len(actual.Outputs.Artifacts)).To(gomega.Equal(len(expected.Outputs.Artifacts)),
        "Output artifact count not matching")

    for key, expectedArtifact := range expected.Outputs.Artifacts {
        actualArtifact, exists := actual.Outputs.Artifacts[key]
        gomega.Expect(exists).To(gomega.BeTrue(), fmt.Sprintf("Output artifact '%s' missing", key))
        MatchArtifacts(actualArtifact, expectedArtifact)

        // For output artifacts, URI should be populated
        gomega.Expect(actualArtifact.Uri).To(gomega.Not(gomega.BeEmpty()),
            fmt.Sprintf("Output artifact '%s' URI should not be empty", key))
    }
}
```

## Phase 4: End-to-End Test Enhancements

### 4.1 Enhanced Pipeline E2E Tests

**File**: `backend/test/end2end/pipeline_e2e_test.go` (Enhancement)

```go
// Add to existing validatePipelineRunSuccess function
func validatePipelineRunSuccess(pipelineFile string, pipelineDir string, testContext *apitests.TestContext) {
    // Existing validation logic...

    // NEW: Enhanced task and artifact validation
    validateTasksAndArtifacts(createdRunID, compiledWorkflow, testContext)
}

// NEW: Validate tasks and their artifacts in completed runs
func validateTasksAndArtifacts(runID string, compiledWorkflow *argo.Workflow, testContext *apitests.TestContext) {
    logger.Log("Validating tasks and artifacts for run %s", runID)

    // Get run with full details
    fullView := v2beta1.GetRunRequest_FULL
    run, err := runClient.GetRun(context.Background(), &v2beta1.GetRunRequest{
        RunId: runID,
        View:  &fullView,
    })
    Expect(err).To(BeNil(), "Failed to get run with full details")

    // Get all tasks for this run
    tasksResponse, err := runClient.ListTasks(context.Background(), &v2beta1.ListTasksRequest{
        RunId:    runID,
        PageSize: 100,
    })
    Expect(err).To(BeNil(), "Failed to list tasks for run")

    // Validate each task has expected artifacts
    for _, task := range tasksResponse.Tasks {
        validateTaskArtifacts(task, compiledWorkflow)
    }
}

// NEW: Validate individual task artifacts
func validateTaskArtifacts(task *v2beta1.PipelineTaskDetail, compiledWorkflow *argo.Workflow) {
    logger.Log("Validating artifacts for task %s", task.TaskName)

    // Find corresponding template in compiled workflow
    template := findWorkflowTemplate(compiledWorkflow, task.TaskName)
    if template == nil {
        logger.Log("No template found for task %s, skipping artifact validation", task.TaskName)
        return
    }

    // Validate output artifacts have URIs populated
    if template.Outputs != nil && len(template.Outputs.Artifacts) > 0 {
        Expect(task.Outputs).To(Not(BeNil()), "Task outputs should not be nil")
        Expect(task.Outputs.Artifacts).To(Not(BeEmpty()), "Task should have output artifacts")

        for artifactName, artifact := range task.Outputs.Artifacts {
            Expect(artifact.Uri).To(Not(BeEmpty()),
                fmt.Sprintf("Output artifact '%s' URI should be populated", artifactName))

            // Validate URI format based on pipeline storage configuration
            validateArtifactURI(artifact.Uri, artifactName)

            // Validate artifact metadata is properly populated
            validateArtifactFields(artifact, task)
        }
    }

    // Validate input artifacts for tasks with dependencies
    if template.Inputs != nil && len(template.Inputs.Artifacts) > 0 {
        Expect(task.Inputs).To(Not(BeNil()), "Task inputs should not be nil")

        for artifactName, artifact := range task.Inputs.Artifacts {
            // Input artifacts should reference existing artifacts
            Expect(artifact.ArtifactId).To(Not(BeEmpty()),
                fmt.Sprintf("Input artifact '%s' should have artifact ID", artifactName))
        }
    }
}

// NEW: Validate artifact URI format and accessibility
func validateArtifactURI(uri string, artifactName string) {
    logger.Log("Validating URI format for artifact %s: %s", artifactName, uri)

    // Validate URI follows expected patterns
    Expect(uri).To(MatchRegexp(`^(gs://|s3://|minio://|file://)`),
        fmt.Sprintf("Artifact URI '%s' should follow valid storage pattern", uri))

    // For object storage URIs, validate path structure
    if strings.HasPrefix(uri, "gs://") || strings.HasPrefix(uri, "s3://") || strings.HasPrefix(uri, "minio://") {
        Expect(uri).To(ContainSubstring(artifactName),
            "Artifact URI should contain artifact name")
    }
}

// NEW: Validate artifact fields are properly set
func validateArtifactFields(artifact *v2beta1.Artifact, task *v2beta1.PipelineTaskDetail) {
    Expect(artifact.ArtifactId).To(Not(BeEmpty()), "Artifact ID should not be empty")
    Expect(artifact.Name).To(Not(BeEmpty()), "Artifact name should not be empty")
    Expect(artifact.Namespace).To(Equal(task.Namespace), "Artifact namespace should match task namespace")
    Expect(artifact.CreatedAt).To(Not(BeNil()), "Artifact creation time should be set")

    // For metric artifacts, validate number value is set
    if artifact.Type == v2beta1.Artifact_Metric {
        Expect(artifact.NumberValue).To(Not(BeNil()), "Metric artifact should have number value")
    }
}
```

### 4.2 E2E Utils Enhancements

**File**: `backend/test/end2end/utils/e2e_utils.go` (Enhancement)

```go
// NEW: Enhanced component status validation including artifacts
func ValidateComponentStatuses(runClient *apiserver.RunClient, k8Client *kubernetes.Clientset,
    testContext *apitests.TestContext, runID string, workflow *argo.Workflow) {

    // Existing validation logic...

    // NEW: Validate artifact creation and relationships
    validateArtifactsForRun(runClient, runID, workflow)
}

// NEW: Comprehensive artifact validation for completed runs
func validateArtifactsForRun(runClient *apiserver.RunClient, runID string, workflow *argo.Workflow) {
    logger.Log("Starting artifact validation for run %s", runID)

    // Get all tasks for the run
    tasksResponse, err := runClient.ListTasks(context.Background(), &v2beta1.ListTasksRequest{
        RunId:    runID,
        PageSize: 100,
    })
    Expect(err).To(BeNil(), "Failed to list tasks")

    // Get all artifacts for the run
    artifactsResponse, err := artifactClient.ListArtifacts(context.Background(), &v2beta1.ListArtifactRequest{
        // Filter by run's namespace
        Namespace: workflow.Namespace,
        PageSize:  100,
    })
    Expect(err).To(BeNil(), "Failed to list artifacts")

    // Get all artifact-task relationships
    artifactTasksResponse, err := artifactClient.ListArtifactTasks(context.Background(), &v2beta1.ListArtifactTasksRequest{
        RunIds:   []string{runID},
        PageSize: 100,
    })
    Expect(err).To(BeNil(), "Failed to list artifact-task relationships")

    // Validate artifact-task relationships are consistent
    validateArtifactTaskConsistency(tasksResponse.Tasks, artifactsResponse.Artifacts, artifactTasksResponse.ArtifactTasks)

    // Validate artifact lineage through the pipeline
    validateArtifactLineage(tasksResponse.Tasks, artifactTasksResponse.ArtifactTasks, workflow)
}

// NEW: Validate artifact-task relationship consistency
func validateArtifactTaskConsistency(tasks []*v2beta1.PipelineTaskDetail, artifacts []*v2beta1.Artifact,
    artifactTasks []*v2beta1.ArtifactTask) {

    // Create lookup maps
    taskMap := make(map[string]*v2beta1.PipelineTaskDetail)
    for _, task := range tasks {
        taskMap[task.TaskId] = task
    }

    artifactMap := make(map[string]*v2beta1.Artifact)
    for _, artifact := range artifacts {
        artifactMap[artifact.ArtifactId] = artifact
    }

    // Validate each artifact-task relationship
    for _, artifactTask := range artifactTasks {
        // Validate referenced task exists
        task, taskExists := taskMap[artifactTask.TaskId]
        Expect(taskExists).To(BeTrue(),
            fmt.Sprintf("Artifact-task references non-existent task: %s", artifactTask.TaskId))

        // Validate referenced artifact exists
        artifact, artifactExists := artifactMap[artifactTask.ArtifactId]
        Expect(artifactExists).To(BeTrue(),
            fmt.Sprintf("Artifact-task references non-existent artifact: %s", artifactTask.ArtifactId))

        // Validate namespace consistency
        Expect(artifact.Namespace).To(Equal(task.Namespace),
            "Artifact and task should be in the same namespace")

        // Validate producer information
        if artifactTask.Type == v2beta1.IOType_OUTPUT {
            Expect(artifactTask.Producer).To(Not(BeNil()), "OUTPUT artifact-task should have producer")
            Expect(artifactTask.Producer.TaskName).To(Equal(task.TaskName),
                "Producer task name should match actual task name")
        }
    }
}

// NEW: Validate artifact lineage through pipeline execution
func validateArtifactLineage(tasks []*v2beta1.PipelineTaskDetail, artifactTasks []*v2beta1.ArtifactTask,
    workflow *argo.Workflow) {

    // Build artifact flow map: artifact_id -> {producing_task, consuming_tasks}
    artifactFlow := buildArtifactFlowMap(artifactTasks)

    // Validate that artifact producers execute before consumers
    for artifactId, flow := range artifactFlow {
        if flow.Producer != nil && len(flow.Consumers) > 0 {
            producerTask := findTaskByName(tasks, flow.Producer.TaskName)
            Expect(producerTask).To(Not(BeNil()),
                fmt.Sprintf("Producer task not found for artifact %s", artifactId))

            for _, consumer := range flow.Consumers {
                consumerTask := findTaskByName(tasks, consumer.TaskName)
                Expect(consumerTask).To(Not(BeNil()),
                    fmt.Sprintf("Consumer task not found for artifact %s", artifactId))

                // Validate producer finished before consumer started
                if producerTask.FinishedAt != nil && consumerTask.StartedAt != nil {
                    producerFinished := time.Unix(producerTask.FinishedAt.Seconds, 0)
                    consumerStarted := time.Unix(consumerTask.StartedAt.Seconds, 0)
                    Expect(producerFinished.Before(consumerStarted)).To(BeTrue(),
                        fmt.Sprintf("Producer task should finish before consumer starts for artifact %s", artifactId))
                }
            }
        }
    }
}
```

## Phase 5: Test Execution Strategy

### 5.1 Test Labels and Organization

```go
// Test execution labels for different test phases
constants.ArtifactAPI      = "ArtifactAPI"
constants.NamespaceIsolation = "NamespaceIsolation"
constants.ArtifactSecurity = "ArtifactSecurity"
constants.ArtifactBulkOps  = "ArtifactBulkOps"
constants.ArtifactE2E      = "ArtifactE2E"
```

### 5.2 Test Execution Commands

```bash
# Run all artifact-related tests
ginkgo -v --label-filter="ArtifactAPI || ArtifactE2E" ./backend/test/v2/api ./backend/test/end2end

# Run only namespace isolation tests
ginkgo -v --label-filter="NamespaceIsolation" ./backend/test/v2/api

# Run artifact security tests
ginkgo -v --label-filter="ArtifactSecurity" ./backend/test/v2/api

# Run full artifact test suite
ginkgo -v --label-filter="Artifact" ./backend/test/...
```

### 5.3 CI/CD Integration

Add to `.github/workflows/` for automated testing:

```yaml
- name: Run Artifact Store Integration Tests
  run: |
    ginkgo -v --label-filter="ArtifactAPI" \
      --junit-report=artifact-api-tests.xml \
      ./backend/test/v2/api

- name: Run Artifact E2E Tests
  run: |
    ginkgo -v --label-filter="ArtifactE2E" \
      --junit-report=artifact-e2e-tests.xml \
      ./backend/test/end2end
```

## Phase 6: Test Data and Fixtures

### 6.1 Test Artifact Definitions

Create test fixtures in `backend/test/testdata/`:

```yaml
# artifact_test_cases.yaml
artifacts:
  basic_dataset:
    type: "Dataset"
    name: "test-dataset"
    uri: "gs://test-bucket/dataset.csv"

  metric_artifact:
    type: "Metric"
    name: "accuracy-score"
    number_value: 0.95

  classification_metrics:
    type: "ClassificationMetric"
    name: "model-metrics"
    metadata:
      accuracy: 0.95
      precision: 0.92
      recall: 0.89
```

### 6.2 Multi-Namespace Test Setup

```go
// Test namespace isolation with multiple users
func setupMultiNamespaceTest() (user1Namespace, user2Namespace string) {
    user1Namespace = "test-user1-" + randomName
    user2Namespace = "test-user2-" + randomName

    // Create test namespaces with proper RBAC
    createTestNamespaceWithRBAC(user1Namespace, "user1")
    createTestNamespaceWithRBAC(user2Namespace, "user2")

    return user1Namespace, user2Namespace
}
```

## Success Criteria

### 1. **API Functionality Coverage**
- ✅ All Artifact CRUD operations tested
- ✅ Bulk operations validated
- ✅ Artifact-Task relationships verified
- ✅ Error handling and edge cases covered

### 2. **Security and Isolation**
- ✅ Namespace isolation enforced
- ✅ RBAC authorization validated
- ✅ Multi-tenant security verified
- ✅ Input validation secured

### 3. **Performance and Reliability**
- ✅ Bulk operations perform within acceptable limits
- ✅ Concurrent access handled correctly
- ✅ Transaction integrity maintained
- ✅ Database query optimization validated

### 4. **End-to-End Validation**
- ✅ Artifact URIs populated correctly
- ✅ Artifact lineage maintained through pipeline execution
- ✅ Task-artifact relationships consistent
- ✅ Metadata preservation validated

## Implementation Timeline

| Phase | Duration | Deliverables |
|-------|----------|-------------|
| Phase 1 | 2 days | Test infrastructure and context setup |
| Phase 2 | 4 days | Core artifact store API tests |
| Phase 3 | 2 days | Validation matchers implementation |
| Phase 4 | 3 days | End-to-end test enhancements |
| Phase 5 | 1 day | Test execution and CI integration |
| Phase 6 | 2 days | Test data, documentation and cleanup |

**Total Estimated Duration**: 14 days

## Conclusion

This comprehensive testing plan ensures the MLMD removal implementation maintains functional parity while improving performance and operational characteristics. The test suite validates critical security boundaries, verifies artifact handling consistency, and provides confidence for production deployment.