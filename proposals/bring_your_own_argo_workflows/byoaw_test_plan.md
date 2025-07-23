# Test Plan: Bring Your Own Argo Workflows (BYOAW)

## Overview

This test plan covers the validation of the "Bring Your Own Argo Workflows" feature, which allows Data Science Pipelines to work with existing Argo Workflows installations instead of deploying their own WorkflowControllers.

## Test Scope

### In Scope
- Global configuration to disable DSP-managed WorkflowControllers
- Compatibility with external Argo Workflows installations
- Version compatibility matrix validation (N and N-1 versions)
- Migration scenarios between DSP-managed and external Argo
- Conflict detection and resolution
- DSPA lifecycle management with external Argo
- Upgrade scenarios
- Security and isolation validation

### Out of Scope
- Partial ArgoWF installs in combination with DSP-shipped Workflow Controller
- Isolating DSP ArgoWF WC from vanilla cluster-scoped ArgoWF installation

## Test Environment Requirements

### Prerequisites
- OpenShift/Kubernetes cluster with RHOAI installed
- Test clusters with various Argo Workflows versions (N and N-1)
- Access to modify DataScienceCluster and DSPA configurations
- Sample pipelines and workflows for testing

### Test Data
- Sample KFP pipelines of varying complexity
- Test workflows with different resource requirements
- Pipeline artifacts and metadata for migration testing

## Test Categories

## 1. Cluster Configuration Tests

### 1.1 Global Configuration Toggle

**Test Case ID**: CC-001
**Test Case Summary**: Verify global toggle to disable WorkflowControllers works correctly
**Test Steps**:
1. Install RHOAI with default configuration (WorkflowControllers enabled)
2. Create a DSPA and verify WorkflowController is deployed
3. Update DataScienceCluster to disable WorkflowControllers globally:
   ```yaml
   spec:
     components:
       datasciencepipelines:
         managementState: Managed
         argoWorkflowsControllers:
           managementState: Removed
   ```
4. Verify existing WorkflowControllers are removed from all DSPAs
5. Create a new DSPA and verify no WorkflowController is deployed

**Expected Results**:
- Global toggle successfully disables WorkflowController deployment
- Existing WorkflowControllers are properly removed
- New DSPAs respect the global configuration
- No data loss occurs during WorkflowController removal

### 1.2 Pre-existing Argo Workflows Detection

**Test Case ID**: CC-002
**Test Case Summary**: Verify behavior when Argo Workflows already exists on cluster
**Test Steps**:
1. Install standalone Argo Workflows on cluster
2. Install RHOAI with BYOAW feature enabled
3. Create DSPA with WorkflowController disabled
4. Verify DSPA components work with external Argo

**Expected Results**:
- Pre-existing Argo CRDs and CRs remain intact
- DSPA deploys successfully without conflicts
- Pipelines execute using external Argo Workflows

## 2. Compatibility Matrix Tests

### 2.1 Current Version Compatibility (N)

**Test Case ID**: CM-001
**Test Case Summary**: Validate compatibility with current supported Argo version
**Test Steps**:
1. Install Argo Workflows version X.Y.Z (current supported version)
2. Configure DSPA to use external Argo
3. Execute test pipeline suite covering:
   - Simple linear pipelines
   - Complex DAG pipelines
   - Pipelines with conditional execution
   - Pipelines with parallel execution
   - Pipelines with artifacts and parameters
4. Verify all pipeline features work correctly

**Expected Results**:
- All pipeline types execute successfully
- Pipeline status reporting works correctly
- Artifacts and metadata are properly handled
- No compatibility issues observed

### 2.2 Previous Version Compatibility (N-1)

**Test Case ID**: CM-002
**Test Case Summary**: Validate compatibility with previous supported Argo version
**Test Steps**:
1. Install Argo Workflows version X.Y-1.Z (previous supported version)
2. Repeat steps from CM-001
3. Document any limitations or known issues

**Expected Results**:
- Core pipeline functionality works with previous version
- Any limitations are documented and acceptable
- No critical failures or data corruption

### 2.3 Version Matrix Validation

**Test Case ID**: CM-003
**Test Case Summary**: Systematically validate compatibility matrix
**Test Steps**:
1. For each supported Argo version in compatibility matrix:
   a. Deploy Argo Workflows version
   b. Configure DSPA for external Argo
   c. Execute standard test pipeline suite
   d. Document results and any issues
2. Update compatibility matrix based on test results

**Expected Results**:
- Compatibility matrix accurately reflects tested compatibility
- All supported versions work as documented
- Unsupported combinations are clearly identified

## 3. Positive Functional Tests

### 3.1 Basic Pipeline Execution

**Test Case ID**: PF-001
**Test Case Summary**: Verify basic pipeline execution with external Argo
**Test Steps**:
1. Configure DSPA with external Argo Workflows
2. Submit simple pipeline (e.g., addition pipeline)
3. Monitor execution through DSP UI
4. Verify completion and results

**Expected Results**:
- Pipeline submits successfully
- Execution progresses normally
- Results are available through DSP interface
- Logs and monitoring work correctly

### 3.2 Complex Pipeline Features

**Test Case ID**: PF-002
**Test Case Summary**: Verify advanced pipeline features work with external Argo
**Test Steps**:
1. Execute pipelines with:
   - Conditional branching
   - Parallel execution
   - Loop constructs
   - Custom components
   - Large data artifacts
2. Verify each feature works correctly

**Expected Results**:
- All advanced pipeline features function correctly
- Performance is acceptable
- Resource utilization is appropriate

### 3.3 Multi-DSPA Environment

**Test Case ID**: PF-003
**Test Case Summary**: Verify multiple DSPAs can share external Argo
**Test Steps**:
1. Create multiple DSPAs in different namespaces
2. Configure all to use same external Argo
3. Execute pipelines simultaneously from different DSPAs
4. Verify isolation and no interference

**Expected Results**:
- Multiple DSPAs work correctly with shared Argo
- Proper namespace isolation maintained
- No cross-contamination of pipelines or data

## 4. Negative Functional Tests

### 4.1 Conflicting WorkflowController Detection

**Test Case ID**: NF-001
**Test Case Summary**: Verify behavior with conflicting WorkflowController configurations
**Test Steps**:
1. Deploy DSPA with WorkflowController enabled
2. Install external Argo Workflows on cluster
3. Attempt to execute pipelines
4. Document behavior and any conflicts

**Expected Results**:
- System behavior is predictable and documented
- No data corruption or loss occurs
- Appropriate warnings or errors are displayed

### 4.2 Incompatible Argo Version

**Test Case ID**: NF-002
**Test Case Summary**: Verify behavior with unsupported Argo version
**Test Steps**:
1. Install unsupported Argo Workflows version
2. Configure DSPA to use external Argo
3. Attempt pipeline execution
4. Document failures and error messages

**Expected Results**:
- Clear error messages indicate incompatibility
- System fails gracefully without corruption
- Appropriate guidance provided to user

### 4.3 Missing External Argo

**Test Case ID**: NF-003
**Test Case Summary**: Verify behavior when external Argo is unavailable
**Test Steps**:
1. Configure DSPA for external Argo (disabled internal WC)
2. Remove or stop external Argo Workflows
3. Attempt pipeline submission
4. Restore external Argo and verify recovery

**Expected Results**:
- Clear error messages when Argo unavailable
- Graceful recovery when Argo restored
- No permanent data loss or corruption

## 5. Migration and Upgrade Tests

### 5.1 DSP-Managed to External Argo Migration

**Test Case ID**: MU-001
**Test Case Summary**: Verify migration from DSP-managed to external Argo
**Test Steps**:
1. Create DSPA with internal WorkflowController
2. Execute several pipelines and verify data/artifacts
3. Install external Argo Workflows
4. Update global configuration to disable internal WCs
5. Verify existing pipeline data remains accessible
6. Execute new pipelines using external Argo

**Expected Results**:
- Migration completes without data loss
- Historical pipeline data remains accessible
- New pipelines work with external Argo
- Artifacts and metadata preserved

### 5.2 External to DSP-Managed Migration

**Test Case ID**: MU-002
**Test Case Summary**: Verify migration from external to DSP-managed Argo
**Test Steps**:
1. Configure DSPA with external Argo
2. Execute pipelines and verify data
3. Update configuration to enable internal WC
4. Remove external Argo configuration
5. Verify pipeline data accessibility and new execution

**Expected Results**:
- Migration to internal WC works correctly
- Pipeline history preserved
- New pipelines execute with internal WC

### 5.3 RHOAI Upgrade with External Argo

**Test Case ID**: MU-003
**Test Case Summary**: Verify RHOAI upgrade preserves external Argo configuration
**Test Steps**:
1. Configure RHOAI with external Argo
2. Execute baseline pipeline tests
3. Upgrade RHOAI to newer version
4. Verify external Argo configuration preserved
5. Re-execute pipeline tests

**Expected Results**:
- Upgrade preserves BYOAW configuration
- External Argo continues to work after upgrade
- No regression in functionality

## 6. Security and Isolation Tests

### 6.1 Namespace Isolation

**Test Case ID**: SI-001
**Test Case Summary**: Verify proper namespace isolation with external Argo
**Test Steps**:
1. Create DSPAs in multiple namespaces
2. Configure each with different service accounts and RBAC
3. Execute pipelines from each namespace
4. Verify no cross-namespace access to pipelines/data

**Expected Results**:
- Pipelines respect namespace boundaries
- RBAC properly enforced
- No unauthorized access between namespaces

### 6.2 External Argo RBAC

**Test Case ID**: SI-002
**Test Case Summary**: Verify RBAC integration with external Argo
**Test Steps**:
1. Configure external Argo with custom RBAC
2. Create DSPA with limited service account
3. Attempt pipeline execution
4. Verify RBAC restrictions are enforced

**Expected Results**:
- External Argo RBAC properly integrated
- DSP respects external Argo permissions
- Unauthorized operations properly blocked

## 7. Performance Tests

### 7.1 Pipeline Execution Performance

**Test Case ID**: PT-001
**Test Case Summary**: Compare performance between internal and external Argo
**Test Steps**:
1. Execute identical pipeline suite with internal WC
2. Execute same suite with external Argo
3. Compare execution times, resource usage, and throughput
4. Document performance characteristics

**Expected Results**:
- Performance with external Argo is acceptable
- No significant degradation compared to internal WC
- Resource utilization within expected bounds

### 7.2 Scale Testing

**Test Case ID**: PT-002
**Test Case Summary**: Verify scalability with external Argo
**Test Steps**:
1. Configure external Argo for high-throughput
2. Submit large number of concurrent pipelines
3. Monitor system performance and stability
4. Verify all pipelines complete successfully

**Expected Results**:
- System handles concurrent pipeline load
- No failures due to resource contention
- Acceptable performance degradation under load

## 8. Boundary Tests

### 8.1 Resource Limits

**Test Case ID**: BT-001
**Test Case Summary**: Verify behavior at resource boundaries
**Test Steps**:
1. Configure external Argo with limited resources
2. Submit resource-intensive pipelines
3. Verify appropriate handling of resource constraints
4. Test recovery when resources become available

**Expected Results**:
- Resource limits properly enforced
- Appropriate error messages for resource constraints
- Graceful recovery when resources available

### 8.2 Large Artifact Handling

**Test Case ID**: BT-002
**Test Case Summary**: Verify handling of large pipeline artifacts
**Test Steps**:
1. Execute pipelines with large data artifacts
2. Monitor storage and transfer performance
3. Verify artifact integrity and accessibility

**Expected Results**:
- Large artifacts handled correctly
- No corruption or loss of artifact data
- Acceptable performance for large transfers

## Test Execution Schedule

### Phase 1: Core Functionality (Weeks 1-2)
- Cluster Configuration Tests (CC-001, CC-002)
- Basic Positive Functional Tests (PF-001, PF-002)
- Basic Negative Tests (NF-001, NF-002)

### Phase 2: Compatibility and Migration (Weeks 3-4)
- Compatibility Matrix Tests (CM-001, CM-002, CM-003)
- Migration Tests (MU-001, MU-002, MU-003)

### Phase 3: Advanced Testing (Weeks 5-6)
- Security and Isolation Tests (SI-001, SI-002)
- Performance Tests (PT-001, PT-002)
- Boundary Tests (BT-001, BT-002)

## Test Environment Matrix

| Test Environment | Argo Version | DSP Version | Purpose |
|------------------|--------------|-------------|---------|
| Env-1 | 3.4.16 | Current | N version testing |
| Env-2 | 3.4.15 | Current | N-1 version testing |
| Env-3 | 3.5.x | Current | Future compatibility |
| Env-4 | 3.4.16 | Previous | Upgrade testing |

## Success Criteria

### Must Have
- All positive functional tests pass
- Compatibility matrix validation complete for supported versions
- Migration scenarios work without data loss
- Security isolation properly maintained
- Performance within acceptable bounds

### Should Have
- Negative test scenarios handled gracefully
- Clear error messages for unsupported configurations
- Comprehensive documentation updated
- Upgrade scenarios validated

### Could Have
- Automated detection of conflicting configurations
- Performance optimizations for external Argo scenarios
- Enhanced monitoring and alerting

## Risk Assessment

### High Risk
- Data loss during migration scenarios
- Security vulnerabilities in multi-tenant environments
- Performance degradation with external Argo

### Medium Risk
- Compatibility issues with specific Argo versions
- Complex configuration management
- Upgrade path complications

### Low Risk
- Minor UI/UX issues
- Documentation gaps
- Non-critical performance variations

## Test Deliverables

1. Test execution reports for each phase
2. Updated compatibility matrix with test results
3. Performance benchmark reports
4. Security validation reports
5. Migration procedure documentation
6. Known issues and limitations documentation
7. Final test summary report with recommendations
