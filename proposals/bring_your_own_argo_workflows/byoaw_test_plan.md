# Test Plan: Bring Your Own Argo WorkFlows (BYOAWF)

## Table of Contents
1. [Overview](#overview)
2. [Test Scope](#test-scope)
3. [Test Environment Requirements](#test-environment-requirements)
4. [Test Categories](#test-categories)
5. [Success Criteria](#success-criteria)
6. [Risk Assessment](#risk-assessment)
## Overview

This test plan validates the "Bring Your Own Argo Workflows" feature, which enables Data Science Pipelines to work with existing Argo Workflows installations instead of deploying dedicated WorkflowControllers. The feature includes a global configuration mechanism to disable KFP-managed WorkflowControllers and ensures compatibility with user-provided Argo Workflows.

The plan covers comprehensive testing scenarios including:
- **Co-existence validation** of KFP and external Argo controllers competing for same events
- **Pre-existing Argo detection** and prevention mechanisms
- **CRD update-in-place** functionality and conflict resolution
- **RBAC compatibility** across different permission models (cluster vs namespace level)
- **Workflow schema version compatibility** and API compatibility validation
- **Z-stream (patch) version compatibility** testing
- **Data preservation** for WorkflowTemplates, CronWorkflows, and pipeline data
- **Independent lifecycle management** of ODH and external Argo Workflows installations
- **Project-level access controls** ensuring workflow visibility boundaries
- **Comprehensive migration scenarios** and upgrade path validation

## Test Scope

### In Scope
- Global configuration toggle to disable/enable WorkflowControllers across all KFPs
- Compatibility validation with external Argo Workflows installations
- Version compatibility matrix testing (N and N-1 versions)
- Migration scenarios between KFP-managed and external Argo configurations
- Conflict detection and resolution mechanisms
- Co-existence testing of KFP and external WorkflowControllers competing for same events
- RBAC compatibility across different permission models (cluster vs namespace level)
- Workflow schema version compatibility validation
- KFP lifecycle management with external Argo
- Security and RBAC integration with external Argo
- Performance impact assessment
- Upgrade scenarios for ODH with external Argo
- Hello world pipeline validation in co-existence scenarios

### Out of Scope
- Partial ArgoWF installs combined with KFP-shipped Workflow Controller
- Isolation between KFP ArgoWF WC and vanilla cluster-scoped ArgoWF installation

## Test Environment Requirements

### Prerequisites
- K8s clusters with KFP installed
- Multiple test environments with different Argo Workflows versions
- Sample pipelines covering various complexity levels
- Test data for migration scenarios

### Test Environments
| Environment | Argo Version   | KFP Version | Purpose                       |
|-------------|----------------|-------------|-------------------------------|
| Env-1       | Current(3.7.x) | Current     | N version compatibility       |
| Env-2       | 3.6.x          | Current     | N-1 version compatibility     |
| Env-3       | 3.4.x - 3.5.y  | Previous    | Upgrade scenarios             |

## Test Categories

## 1. Cluster Configuration Tests
This section covers tests for different cluster configurations to ensure BYOAW functionality across various deployment scenarios.

### 1.1 Kubernetes Native Mode

| Test Case ID          | TC-CC-003                                                                                                                                                                                                                           |
|-----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify BYOAW compatibility with Kubernetes Native Mode - Create Pipeline Via CR                                                                                                                                                     |
| **Test Steps**        | <ol><li>Configure cluster for Kubernetes Native Mode</li><li>Install external Argo Workflows</li><li>Disable KFP WorkflowControllers globally</li><li>Create KFP</li><li>Create Pipeline via CR and create a pipeline run</li></ol> |
| **Expected Results**  | <ul><li>Kubernetes Native Mode works with external Argo</li><li> Pipeline execution uses Kubernetes-native constructs</li><li> No conflicts between modes </li>                                                                     |

| Test Case ID          | TC-CC-003                                                                                                                                                                                                                               |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify BYOAW compatibility with Kubernetes Native Mode - Create Pipeline via API                                                                                                                                                        |
| **Test Steps**        | <ol><li>Configure cluster for Kubernetes Native Mode</li><li>Install external Argo Workflows</li><li>Disable KFP WorkflowControllers globally</li><li>Create KFP</li><li>Create Pipeline via API/UI and create a pipeline run</li></ol> |
| **Expected Results**  | <ul><li>Kubernetes Native Mode works with external Argo</li><li> Pipeline executes successfully </li>                                                                                                                                   |

### 1.2 Kubeflow Mode Compatibility

| Test Case ID          | TC-CC-004                                                                                                                                                                                                                    |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify BYOAW works in Kubeflow-enabled clusters                                                                                                                                                                              |
| **Test Steps**        | <ol><li>Configure Kubeflow-enabled cluster</li><li>Install Kubeflow-compatible external Argo</li><li>Configure KFP with external Argo</li><li>Execute pipeline suite</li><li>Verify Kubeflow compliance maintained</li></ol> |
| **Expected Results**  | <ul><li>External Argo respects Kubeflow requirements</li><li> Pipeline execution maintains Kubeflow compliance</li><li> No cryptographic violations </li>                                                                    |                                                      |

## 2. Positive Functional Tests
This section covers all positive functional tests to make sure that feature works as expected and there is no regression as well

### 2.1 Basic Pipeline Execution

| Test Case ID          | TC-PF-001                                                                                                                                                                                                     |
|-----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify basic pipeline execution with external Argo                                                                                                                                                            |
| **Test Steps**        | <ol><li>Configure KFP with external Argo</li><li>Submit simple addition pipeline</li><li> Monitor execution through KFP UI</li><li> Verify completion and results</li><li> Check logs and artifacts</li></ol> |
| **Expected Results**  | <ul><li>Pipeline submits successfully</li><li> Execution progresses normally</li><li> Results accessible through KFP interface</li><li> Logs and monitoring functional </li>                                  |

### 2.2 Complex Pipeline Types
Runs of different types of pipeline specs executes successfully. Pipelines that exercise all different inputs and outputs of a launcher/driver

| Test Case ID          | TC-PF-002                                                                                                                                                                                                                       |
|-----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Pipelines with artifacts" pipeline                                                                                                                                                                               |
| **Test Steps**        | <ol><li>Configure KFP with external Argo </li><li> Execute pipeline - Pipelines with artifacts</li><li> Verify each pipeline type executes correctly</li><li> Validate artifacts, metadata, and custom configurations</li></ol> |
| **Expected Results**  | <ul><li>Pipeline execute successfully</li><li> Artifacts are produced to the right s3 location and are consumed correctly  </li>                                                                                                |

| Test Case ID          | TC-PF-003                                                                                                                                                          |
|-----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Pipelines without artifacts" pipeline                                                                                                               |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Pipelines without artifacts</li><li> Verify each pipeline type executes correctly</li></ol> |                                                                                                                                                                                   |
| **Expected Results**  | <ul><li>Pipeline runs successfully</li><li> No artifacts are produced to S3 </li>                                                                                  |

| Test Case ID          | TC-PF-004                                                                                                                                                  |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "For loop constructs" pipeline                                                                                                               |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - For loop constructs</li><li> Verify each pipeline type executes correctly</li></ol> |                                                                                                                                                                                   |
| **Expected Results**  | <ul><li>Pipeline runs successfully</li><li> DAGs inside the for loop are interated over correctly </li>                                                    |


| Test Case ID          | TC-PF-005                                                                                                                                                     |
|-----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Parallel for execution" pipeline                                                                                                               |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Parallel for execution</li><li> Verify each pipeline type executes correctly</li></ol> |                                                                                                                                                                                   |
| **Expected Results**  | <ul><li>Pipeline runs successfully</li><li> Parallel DAGs running in parallel and completes successfully  </li>                                               |


| Test Case ID          | TC-PF-006                                                                                                                                                                                     |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Custom root KFP components" pipeline                                                                                                                                           |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Custom root KFP components </li><li> Verify each pipeline type executes correctly </li></ol>                           |
| **Expected Results**  | <ul><li>Pipeline runs successfully</li><li> Artifcats are uploaded in the custom S3 bucket rather than the default, and downstream components are consuming from this custom location </li>   |

| Test Case ID          | TC-PF-007                                                                                                                                                              |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Custom python package indexes" pipeline                                                                                                                 |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Custom python package indexes </li><li> Verify each pipeline type executes correctly </li></ol> |
| **Expected Results**  | <ul><li>Pipeline runs successfully</li><li> When driver and launcher downloads python packages, it downloads from the custom index rather than pypi  </li>             |

| Test Case ID          | TC-PF-008                                                                                                                                                                |
|-----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Pipelines with input parameters" pipeline                                                                                                                 |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Pipelines with input parameters </li><li> Verify each pipeline type executes correctly </li></ol> |
| **Expected Results**  | <ul><li>Pipeline runs successfully<br/> Components are consuming the right parameters (verify it in the logs or input resolution in the Argo Workflow Status) </li>      |

| Test Case ID          | TC-PF-009                                                                                                                                                  |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Custom base images" pipeline                                                                                                                |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Custom base images </li><li> Verify each pipeline type executes correctly</li></ol> |
| **Expected Results**  | <ul><li>Pipeline runs successfully</li><li> Components are downloading custom base images </li>                                                            |

| Test Case ID          | TC-PF-010                                                                                                                                                                              |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Pipelines with both input and output artifacts" pipeline                                                                                                                |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Pipelines with both input and output artifacts </li><li> Verify each pipeline type executes correctly</li></ol> |
| **Expected Results**  | <ul><li>Pipeline runs successfully</li><li> Upstream and Downstream components can produce & consume artifacts </li>                                                                   |

| Test Case ID          | TC-PF-011                                                                                                                                                                  |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Pipelines without input parameters" pipeline                                                                                                                |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Pipelines without input parameters </li><li> Verify each pipeline type executes correctly</li></ol> |
| **Expected Results**  | <ul><li>Pipeline runs successfully </li>                                                                                                                                   |

| Test Case ID          | TC-PF-012                                                                                                                                                               |
|-----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Pipelines with NO input artifacts, but just output artifacts" pipeline                                                                                   |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Pipelines with output artifacts </li><li> Verify each pipeline type executes correctly</li></ol> |
| **Expected Results**  | <ul><li>Pipeline runs successfully</li><li> Output artifacts (like a model/trained data) are produced to S3 correctly </li>                                             |

| Test Case ID          | TC-PF-013                                                                                                                                                                  |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Pipelines without output artifacts" pipeline                                                                                                                |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Pipelines without output artifacts </li><li> Verify each pipeline type executes correctly</li></ol> |
| **Expected Results**  | <ul><li>Pipeline runs successfully </li>                                                                                                                                   |

| Test Case ID          | TC-PF-014                                                                                                                                                             |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Pipelines with iteration count" pipeline                                                                                                               |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Pipelines with iteration count </li><li>Verify each pipeline type executes correctly</li></ol> |
| **Expected Results**  | <ul><li>Pipeline runs successfully</li><li> DAGs are iterated over for the correct number of iterations </li>                                                         |

| Test Case ID          | TC-PF-015                                                                                                                                                              |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Pipelines with retry mechanisms" pipeline                                                                                                               |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Pipelines with retry mechanisms </li><li>Verify each pipeline type executes correctly</li></ol> |
| **Expected Results**  | <ul><li>Pipeline runs successfully</li><li> Components are retried the correct number of times in case of any failure </li>                                            |

| Test Case ID          | TC-PF-016                                                                                                                                                                  |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Pipelines with certificate handling" pipeline                                                                                                               |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Pipelines with certificate handling </li><li>Verify each pipeline type executes correctly</li></ol> |
| **Expected Results**  | <ul><li>Pipeline runs successfully</li><li> Components gets the right certificate installed </li>                                                                          |

| Test Case ID          | TC-PF-017                                                                                                                                                              |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify run of "Conditional branching pipelines" pipeline                                                                                                               |
| **Test Steps**        | <ol><li> Configure KFP with external Argo </li><li> Execute Pipeline - Conditional branching pipelines </li><li>Verify each pipeline type executes correctly</li></ol> |
| **Expected Results**  | <ul><li>Pipeline runs successfully</li><li> Nested DAGs runs only if the expected condition is true </li>                                                              |

### 2.3 Pod Spec Override Testing
Tests to validate that if you override Pod Spec, then correct kubernetes properties gets applied when the pods are created

| Test Case ID          | TC-PF-018                                                                                                                                                                                |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify pipeline execution with Pod spec overrides containing "Node taints and tolerations"                                                                                               |
| **Test Steps**        | <ol><li> Configure pipelines with Pod spec patch : - Node taints and tolerations</li><li>Execute pipelines with external Argo  </li></ol>                                                |
| **Expected Results**  | <ul><li>Pod spec overrides applied successfully</li><li> Pipelines schedule on correct nodes</li><li> PVCs mounted and accessible</li><li> Custom labels and annotations present </li>   |

| Test Case ID          | TC-PF-019                                                                                                                                   |
|-----------------------|---------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify pipeline execution with Pod spec overrides containing "Custom labels and annotations"                                                |
| **Test Steps**        | <ol><li> Configure pipelines with Pod spec patch : - Custom labels and annotations </li><li>Execute pipelines with external Argo </li></ol> |
| **Expected Results**  | <ul><li>Pod spec overrides applied successfully</li><li> PVCs mounted and accessible</li><li> Custom labels and annotations present </li>   |

| Test Case ID          | TC-PF-020                                                                                                                                                                                                           |
|-----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify pipeline execution with Pod spec overrides containing "Resource limits"                                                                                                                                      |
| **Test Steps**        | <ol><li> Configure pipelines with Pod spec patch : - Resource limits </li><li>Execute pipelines with external Argo </li></ol>                                                                                       |
| **Expected Results**  | <ul><li>Pod spec overrides applied successfully</li><li> Overridden component pod has the right resource limit assigned</li><li> PVCs mounted and accessible</li><li> Custom labels and annotations present </li>   |

| Test Case ID          | TC-PF-021                                                                                                               |
|-----------------------|-------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify pipeline execution with component using GPU "Set Acceleration type and limit"                                    |
| **Test Steps**        | <ol><li> Configure pipelines with component requesting GPU </li><li>Execute pipelines with external Argo     </li></ol> |
| **Expected Results**  | <ul><li>Pod spec overrides applied successfully</li><li> Overridden component pod has the correct GPU allocated </li>   |

### 2.4 Multi-KFP Environment

| Test Case ID          | TC-PF-022                                                                                                                                                                                                                |
|-----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify multiple KFPs sharing external Argo                                                                                                                                                                               |
| **Test Steps**        | <ol><li> Create KFPs in different namespaces</li><li>Configure all for external Argo</li><li>Execute pipelines simultaneously</li><li>Verify namespace isolation</li><li>Check resource sharing and conflicts </li></ol> |
| **Expected Results**  | <ul><li>Multiple KFPs operate independently</li><li> Proper namespace isolation maintained</li><li> No pipeline interference or data leakage</li><li> Resource sharing works correctly  </li>                            |

## 3. Negative Functional Tests
This section overs error handling scenarios to make sure we are handling non-ideal cases within expectations

### 3.1 Conflicting WorkflowController Detection

| Test Case ID          | TC-NF-001                                                                                                                                                                                                                           |
|-----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify behavior with conflicting WorkflowController configurations                                                                                                                                                                  |
| **Test Steps**        | <ol><li> Deploy KFP with WorkflowController enabled</li><li>Install external Argo on same cluster</li><li>Attempt pipeline execution</li><li>Document conflicts and behavior</li><li>Test conflict resolution mechanisms </li></ol> |
| **Expected Results**  | <ul><li>System behavior is predictable</li><li> Appropriate warnings displayed</li><li> No data corruption</li><li> Clear guidance provided </li>                                                                                   |

### 3.1.1 Co-existing WorkflowController Event Conflicts

| Test Case ID          | TC-NF-001a                                                                                                                                                                                                                                                                                                                                            |
|-----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Test KFP and External WorkflowControllers co-existing and competing for same events                                                                                                                                                                                                                                                                   |
| **Test Steps**        | <ol><li> Deploy KFP with internal WorkflowController</li><li>Install external Argo WorkflowController watching same namespaces</li><li>Submit pipeline that creates Workflow CRs</li><li>Monitor which controller processes the workflow</li><li>Verify event handling and potential conflicts</li><li>Test resource ownership and cleanup </li></ol> |
| **Expected Results**  | <ul><li>Event conflicts properly identified</li><li> Clear ownership of workflow resources</li><li> No orphaned or stuck workflows</li><li> Predictable controller behavior documented </li>                                                                                                                                                          |

### 3.2 Incompatible Argo Version

| Test Case ID          | TC-NF-002                                                                                                                                                                                           |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify behavior with unsupported Argo versions                                                                                                                                                      |
| **Test Steps**        | <ol><li> Install unsupported Argo version</li><li>Configure KFP for external Argo</li><li>Attempt pipeline execution</li><li>Document error messages</li><li>Verify graceful degradation </li></ol> |
| **Expected Results**  | <ul><li>Clear incompatibility errors</li><li> Graceful failure without corruption</li><li> Helpful guidance for resolution </li>                                                                    |

### 3.3 Missing External Argo

| Test Case ID          | TC-NF-003                                                                                                                                                                                               |
|-----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify behavior when external Argo unavailable                                                                                                                                                          |
| **Test Steps**        | <ol><li> Configure KFP for external Argo</li><li>Stop/remove external Argo service</li><li>Attempt pipeline submission</li><li>Restore Argo and verify recovery</li><li>Check data integrity </li></ol> |
| **Expected Results**  | <ul><li>Clear error messages when Argo unavailable</li><li> Graceful recovery when restored</li><li> No permanent data loss </li>                                                                       |

### 3.4 Invalid Pipeline Submissions

| Test Case ID          | TC-NF-004                                                                                                                                                                                   |
|-----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Test invalid pipeline handling with external Argo                                                                                                                                           |
| **Test Steps**        | <ol><li> Submit pipelines from `data/pipeline_files/invalid/`</li><li>Verify appropriate error handling</li><li>Check error message clarity</li><li>Ensure no system instability </li></ol> |
| **Expected Results**  | <ul><li>Invalid pipelines rejected appropriately</li><li> Clear error messages provided</li><li> System remains stable</li><li> No resource leaks </li>                                     |

### 3.5 Unsupported Configuration Detection

| Test Case ID          | TC-NF-005                                                                                                                                                                                                                                                                                                            |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify detection of unsupported individual KFP WorkflowController disable                                                                                                                                                                                                                                            |
| **Test Steps**        | <ol><li> Set global WorkflowController management to Removed</li><li>Attempt to create KFP with individual `workflowController.deploy: false`</li><li>Verify appropriate warning/error messages</li><li>Test documentation guidance for users</li><li>Ensure configuration is flagged as development-only </li></ol> |
| **Expected Results**  | <ul><li>Unsupported configuration detected</li><li> Clear warning messages displayed</li><li> Documentation provides proper guidance</li><li> Development-only usage clearly indicated </li>                                                                                                                         |

### 3.6 CRD Version Conflicts

| Test Case ID          | TC-NF-006                                                                                                                                                                                                                                           |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Test behavior with conflicting Argo CRD versions                                                                                                                                                                                                    |
| **Test Steps**        | <ol><li> Install KFP with specific Argo CRD version</li><li>Install external Argo with different CRD version</li><li>Attempt pipeline execution</li><li>Verify conflict detection and resolution</li><li>Test update-in-place mechanisms </li></ol> |
| **Expected Results**  | <ul><li>CRD version conflicts detected</li><li> Update-in-place works when compatible</li><li> Clear error messages for incompatible versions</li><li> No existing workflow corruption </li>                                                        |

### 3.7 Different RBAC Between KFP and External Argo

| Test Case ID          | TC-NF-007                                                                                                                                                                                                                                                                                                                                          |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Test KFP and external WorkflowController with different RBAC configurations                                                                                                                                                                                                                                                                        |
| **Test Steps**        | <ol><li> Configure KFP with cluster-level RBAC permissions</li><li>Install external Argo with namespace-level RBAC restrictions</li><li>Submit pipelines through KFP interface</li><li>Verify RBAC conflicts and permission issues</li><li>Test resource access and execution failures</li><li>Document RBAC compatibility requirements </li></ol> |
| **Expected Results**  | <ul><li>RBAC conflicts properly identified</li><li> Clear error messages for permission issues</li><li> Guidance provided for RBAC alignment</li><li> No security violations or escalations </li>                                                                                                                                                  |

### 3.8 KFP with Incompatible Workflow Schema

| Test Case ID          | TC-NF-008                                                                                                                                                                                                                                                                                                         |
|-----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Test KFP behavior with incompatible workflow schema versions                                                                                                                                                                                                                                                      |
| **Test Steps**        | <ol><li> Install external Argo with older workflow schema</li><li>Configure KFP to use external Argo</li><li>Submit pipelines with newer schema features</li><li>Verify schema compatibility checking</li><li>Test graceful degradation or error handling</li><li>Document schema compatibility matrix </li></ol> |
| **Expected Results**  | <ul><li>Schema incompatibilities detected</li><li> Clear error messages about schema conflicts</li><li> Graceful handling of unsupported features</li><li> No workflow corruption or data loss </li>                                                                                                              |

## 4. Boundary Tests
Type of performance test to confirm that our current limits to resources and artifacts are still handled properly

### 4.1 Resource Limits

| Test Case ID          | TC-BT-001                                                                                                                                                                                                                             |
|-----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify behavior at resource boundaries                                                                                                                                                                                                |
| **Test Steps**        | <ol><li> Configure external Argo with resource limits</li><li>Submit resource-intensive pipelines</li><li>Monitor resource utilization</li><li>Verify appropriate throttling</li><li>Test recovery when resources available</li></ol> |
| **Expected Results**  | <ul><li>Resource limits properly enforced</li><li> Appropriate queuing/throttling behavior</li><li> Clear resource constraint messages</li><li> Graceful recovery when resources free </li>                                           |

### 4.2 Large Artifact Handling

| Test Case ID          | TC-BT-002                                                                                                                                                                                                              |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify handling of large pipeline artifacts                                                                                                                                                                            |
| **Test Steps**        | <ol><li> Configure pipelines with large data artifacts</li><li>Execute with external Argo</li><li>Monitor storage and transfer performance</li><li>Verify artifact integrity</li><li>Test cleanup mechanisms</li></ol> |
| **Expected Results**  | <ul><li>Large artifacts handled efficiently</li><li> No data corruption or loss</li><li> Acceptable transfer performance</li><li> Proper cleanup after completion </li>                                                |

### 4.3 High Concurrency

| Test Case ID          | TC-BT-003                                                                                                                                                                                                         |
|-----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Test high concurrency scenarios                                                                                                                                                                                   |
| **Test Steps**        | <ol><li> Submit multiple concurrent pipelines</li><li>Monitor external Argo performance</li><li>Verify all pipelines complete</li><li>Check for resource contention</li><li>Validate result consistency</li></ol> |
| **Expected Results**  | <ul><li>High concurrency handled appropriately</li><li> No pipeline failures due to contention</li><li> Consistent execution results</li><li> Stable system performance </li>                                     |

## 5. Performance Tests
Load Testing - this is just to make sure that with the change in argo workflow, there is no impact on the performance of components that are under our control

### 5.1 Execution Performance Comparison

| Test Case ID          | TC-PT-001                                                                                                                                                                                                                                                 |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Compare performance between internal and external Argo                                                                                                                                                                                                    |
| **Test Steps**        | <ol><li> Execute identical pipeline suite with internal WC</li><li>Execute same suite with external Argo</li><li>Measure execution times and resource usage</li><li>Compare throughput and latency</li><li>Document performance characteristics</li></ol> |
| **Expected Results**  | <ul><li>Performance with external Argo acceptable</li><li> No significant degradation vs internal WC</li><li> Resource utilization within bounds</li><li> Scalability maintained </li>                                                                    |

### 5.2 Startup and Initialization

| Test Case ID          | TC-PT-002                                                                                                                                                                                                                                 |
|-----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Measure KFP startup time with external Argo                                                                                                                                                                                               |
| **Test Steps**        | <ol><li> Measure KFP creation time with internal WC</li><li>Measure KFP creation time with external Argo</li><li>Compare initialization times</li><li>Monitor resource usage during startup</li><li>Document timing differences</li></ol> |
| **Expected Results**  | <ul><li>Startup time with external Argo reasonable</li><li> Initialization completes successfully</li><li> Resource usage during startup acceptable</li><li> No significant delays </li>                                                  |

## 6. Compatibility Matrix Tests
Based on the compatability matrix as defined in #Test Environments

### 6.1 Current Version (N) Compatibility

| Test Case ID          | TC-CM-001                                                                                                                                                                                                                                      |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Validate compatibility with current supported Argo version                                                                                                                                                                                     |
| **Test Steps**        | <ol><li> Install current supported Argo version (e.g., 3.4.16)</li><li>Configure KFP for external Argo</li><li>Execute comprehensive pipeline test suite</li><li>Verify all features work correctly</li><li>Document any limitations</li></ol> |
| **Expected Results**  | <ul><li>Full compatibility with current version</li><li> All pipeline features operational</li><li> No breaking changes or issues</li><li> Performance within acceptable range </li>                                                           |

### 6.2 Previous Version (N-1) Compatibility

| Test Case ID          | TC-CM-002                                                                                                                                                                                                                                                   |
|-----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Validate compatibility with previous supported Argo version                                                                                                                                                                                                 |
| **Test Steps**        | <ol><li> Install previous supported Argo version (e.g., 3.4.15)</li><li>Configure KFP for external Argo</li><li>Execute comprehensive pipeline test suite</li><li>Document compatibility differences</li><li>Verify core functionality maintained</li></ol> |
| **Expected Results**  | <ul><li>Core functionality works with N-1 version</li><li> Any limitations clearly documented</li><li> No critical failures or data loss</li><li> Upgrade path available  </li>                                                                             |

### 6.2.1 Z-Stream Version Compatibility

| Test Case ID          | TC-CM-002a                                                                                                                                                                                                                                                                                                                                                                 |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Validate compatibility with z-stream (patch) versions of Argo                                                                                                                                                                                                                                                                                                              |
| **Test Steps**        | <ol><li> Test current KFP with multiple z-stream versions of same minor Argo release</li><li>Example: Test KFP v3.4.17 with Argo v3.4.16, v3.4.17, v3.4.18</li><li>Execute standard pipeline test suite for each z-stream version</li><li>Document any breaking changes in patch versions</li><li>Verify backward and forward compatibility within minor version</li></ol> |
| **Expected Results**  | <ul><li>Z-stream versions maintain compatibility</li><li> No breaking changes in patch releases</li><li> Smooth operation across patch versions</li><li> Clear documentation of any exceptions </li>                                                                                                                                                                       |

### 6.3 Version Matrix Validation

| Test Case ID          | TC-CM-003                                                                                                                                                                                                                                                                             |
|-----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Systematically validate compatibility matrix                                                                                                                                                                                                                                          |
| **Test Steps**        | <ol><li> For each version in compatibility matrix:<br/>   a. Deploy specific Argo version<br/>   b. Configure KFP<br/>   c. Execute standard test suite<br/>   d. Document results and issues</li><li>Update compatibility matrix</li><li>Identify unsupported combinations</li></ol> |
| **Expected Results**  | <ul><li>Compatibility matrix accurately reflects reality</li><li> All supported versions documented</li><li> Unsupported combinations identified</li><li> Clear guidance for version selection  </li>                                                                                 |

### 6.4 KFP and External Argo Co-existence Validation

| Test Case ID          | TC-CM-004                                                                                                                                                                                                                                                                                                                                                                                                          |
|-----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Validate successful hello world pipeline with KFP and External Argo co-existing                                                                                                                                                                                                                                                                                                                                    |
| **Test Steps**        | <ol><li> Deploy KFP with internal WorkflowController</li><li>Install external Argo WorkflowController on same cluster</li><li>Submit simple hello world pipeline through KFP</li><li>Verify pipeline executes successfully using KFP controller</li><li>Verify external Argo remains unaffected</li><li>Test pipeline monitoring and status reporting</li><li>Validate artifact handling and logs access</li></ol> |
| **Expected Results**  | <ul><li>Hello world pipeline executes successfully</li><li> KFP WorkflowController processes the pipeline</li><li> External Argo WorkflowController unaffected</li><li> No resource conflicts or interference</li><li> Pipeline status and logs accessible</li><li> Artifacts properly stored and retrievable </li>                                                                                                |

### 6.5 API Server and WorkflowController Compatibility

| Test Case ID          | TC-CM-005                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify KFP API Server compatibility with different external WorkflowController versions                                                                                                                                                                                                                                                                                                                                                                              |
| **Test Steps**        | <ol><li> Deploy KFP API Server with specific Argo library dependencies</li><li>Install external Argo WorkflowController with different version</li><li>Test API Server to WorkflowController communication</li><li>Verify Kubernetes API interactions (CRs, status updates)</li><li>Test pipeline submission, execution, and status reporting</li><li>Monitor for API compatibility issues or version mismatches</li><li>Document API compatibility matrix</li></ol> |
| **Expected Results**  | <ul><li>API Server communicates successfully with external WC</li><li> Kubernetes API interactions work correctly</li><li> Pipeline lifecycle management functions properly</li><li> Status updates and monitoring work correctly</li><li> API compatibility documented and validated  </li>                                                                                                                                                                         |

## 7. Uninstall and Data Preservation Tests
Verify that if you uninstall KFP or Argo Workflow Controller, then the data is still preserved, so that the next time deployment happens, things continue - this includes use case for different deployment strategies

### 7.1 KFP Uninstall with External Argo

| Test Case ID          | TC-UP-001                                                                                                                                                                                                                                                                                                                      |
|-----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify KFP uninstall behavior with external Argo                                                                                                                                                                                                                                                                               |
| **Test Steps**        | <ol><li> Configure KFP with external Argo (no internal WC)</li><li>Execute multiple pipelines and generate data</li><li>Delete KFP</li><li>Verify external Argo WorkflowController remains intact</li><li>Verify KFP-specific resources are cleaned up</li><li>Check that pipeline history is appropriately handled </li></ol> |
| **Expected Results**  | <ul><li>KFP removes cleanly</li><li> External Argo WorkflowController unaffected</li><li> No impact on other KFPs using same external Argo</li><li> Pipeline data handling follows standard procedures  </li>                                                                                                                  |

### 7.2 KFP Uninstall with Internal WorkflowController

| Test Case ID          | TC-UP-002                                                                                                                                                                                                                                                                         |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify standard KFP uninstall with internal WorkflowController                                                                                                                                                                                                                    |
| **Test Steps**        | <ol><li> Configure KFP with internal WorkflowController</li><li>Execute pipelines and generate data</li><li>Delete KFP</li><li>Verify WorkflowController is removed with KFP</li><li>Verify proper cleanup of all KFP components</li><li>Ensure no external Argo impact</li></ol> |
| **Expected Results**  | <ul><li>KFP and WorkflowController removed completely</li><li> Standard cleanup procedures followed</li><li> No resource leaks or orphaned components</li><li> External Argo installations unaffected </li>                                                                       |

### 7.3 Data Preservation During WorkflowController Transitions

| Test Case ID          | TC-UP-003                                                                                                                                                                                                                                                                                                                                                         |
|-----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify data preservation during WorkflowController management transitions                                                                                                                                                                                                                                                                                         |
| **Test Steps**        | <ol><li> Create KFP with internal WC and execute pipelines</li><li>Disable WC globally (transition to external Argo)</li><li>Verify run history, artifacts, and metadata preserved</li><li>Re-enable WC globally (transition back to internal)</li><li>Verify all historical data remains accessible</li><li>Test new pipeline execution in both states</li></ol> |
| **Expected Results**  | <ul><li>Pipeline run history preserved across transitions</li><li> Artifacts remain accessible</li><li> Metadata integrity maintained</li><li> New pipelines work in both configurations </li>                                                                                                                                                                    |

### 7.4 WorkflowTemplates and CronWorkflows Preservation

| Test Case ID          | TC-UP-004                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify preservation of WorkflowTemplates and CronWorkflows during KFP install/uninstall                                                                                                                                                                                                                                                                                                                                                                   |
| **Test Steps**        | <ol><li> Install external Argo and create WorkflowTemplates and CronWorkflows</li><li>Install KFP with BYOAW configuration</li><li>Verify existing WorkflowTemplates and CronWorkflows remain intact</li><li>Create additional WorkflowTemplates through KFP interface</li><li>Uninstall KFP components</li><li>Verify all WorkflowTemplates and CronWorkflows still exist</li><li>Test functionality of preserved resources with external Argo</li></ol> |
| **Expected Results**  | <ul><li>Pre-existing WorkflowTemplates and CronWorkflows preserved</li><li> KFP-created templates also preserved during uninstall</li><li> All preserved resources remain functional</li><li> No data corruption or resource deletion</li><li> External Argo can use all preserved templates </li>                                                                                                                                                        |

## 8. Migration and Upgrade Tests
Covers migration from internal to external WC and vice versa. Also covers upgrade of ODH and Argo versions

### 8.1 KFP Upgrade Scenarios

| Test Case ID          | TC-MU-003                                                                                                                                                                                                            |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify KFP upgrade preserves external Argo setup                                                                                                                                                                     |
| **Test Steps**        | <ol><li> Configure KFP with external Argo</li><li>Execute baseline pipeline tests</li><li>Upgrade KFP to newer version</li><li>Verify external Argo configuration intact</li><li>Re-execute pipeline tests</li></ol> |
| **Expected Results**  | <ul><li>Upgrade preserves BYOAW configuration</li><li> External Argo continues working</li><li> No functionality regression</li><li> Configuration settings maintained  </li>                                        |

### 8.2 Argo Version Upgrade with External Installation

| Test Case ID          | TC-MU-004                                                                                                                                                                                                                                                                               |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Verify external Argo version upgrade scenarios                                                                                                                                                                                                                                          |
| **Test Steps**        | <ol><li> Configure KFP with external Argo version N-1</li><li>Execute baseline pipeline tests</li><li>Upgrade external Argo to version N</li><li>Verify compatibility matrix adherence</li><li>Test pipeline execution post-upgrade</li><li>Document any required ODH updates</li></ol> |
| **Expected Results**  | <ul><li>External Argo upgrade completes successfully</li><li> Compatibility maintained within support matrix</li><li> Clear guidance for required ODH updates</li><li> Pipeline functionality preserved </li>                                                                           |


## Success Criteria

### Must Have
- All positive functional tests pass without failures
- Compatibility matrix validation complete for N and N-1 versions
- Z-stream (patch) version compatibility validated
- Migration scenarios preserve data integrity
- Security and RBAC properly enforced
- Performance within acceptable bounds (no >20% degradation)
- Platform-level CRD and RBAC management works correctly
- Data preservation during WorkflowController transitions
- Sub-component removal functionality validated
- Pre-existing Argo detection and prevention working
- CRD update-in-place functionality validated
- WorkflowTemplates and CronWorkflows preservation confirmed
- API Server to WorkflowController compatibility verified
- Workflow visibility and project access controls enforced

### Should Have
- Negative test scenarios handled gracefully
- Clear error messages for all failure modes
- Unsupported configuration detection functional
- CRD version conflict resolution working
- RBAC conflict detection and resolution
- Schema compatibility validation working
- Co-existence scenarios validated successfully
- Independent lifecycle management validated
- Documentation complete and accurate
- Uninstall scenarios preserve external Argo integrity

### Could Have
- Performance optimizations for external Argo scenarios
- Enhanced monitoring and observability
- Additional version compatibility beyond N-1
- Automated detection of conflicting configurations
- Advanced CRD update-in-place mechanisms

## Risk Assessment

### High Risk
- Data loss during migration scenarios
- Security vulnerabilities in multi-tenant setups
- Performance degradation with external Argo
- Incompatibility with future Argo versions

### Medium Risk
- Complex configuration management
- Upgrade complications
- Resource contention in shared scenarios
- Error handling gaps

### Low Risk
- Minor UI/UX inconsistencies
- Documentation completeness
- Non-critical performance variations
- Edge case handling