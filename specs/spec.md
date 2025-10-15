# Test Infrastructure Modernization and Consolidation

## Feature Overview

This feature aims to comprehensively refactor and modernize the Kubeflow Pipelines (KFP) test infrastructure by:
- Converting all Go tests from standard testing library to Ginkgo+Gomega BDD framework
- Migrating all Python tests to use pytest framework consistently
- Consolidating scattered Python test files into a unified `sdk/tests` package structure
- Eliminating duplicate test code and establishing reusable test utilities
- Improving test maintainability, readability, and developer experience

The modernized test infrastructure will provide a consistent testing experience across the entire KFP codebase, reduce maintenance overhead, improve test reliability, and establish best practices for future test development.

## Goals

1. **Framework Standardization**: Migrate all Go tests to Ginkgo+Gomega and all Python tests to pytest
2. **Code Consolidation**: Eliminate duplicate test code and create reusable test utilities
3. **Structural Reorganization**: Consolidate all Python SDK tests into `sdk/tests` package with clear categorization
4. **Improved Maintainability**: Establish consistent test patterns that are easier to understand and maintain
5. **Enhanced Developer Experience**: Provide better test output, clearer failure messages, and improved debugging capabilities
6. **Test Coverage Preservation**: Maintain or improve existing test coverage during migration
7. **CI/CD Integration**: Ensure seamless integration with existing CI/CD pipelines without disruption

## Out of Scope

- Adding new test coverage for previously untested features (focus is on migration/refactoring existing tests)
- Performance optimization of test execution time (beyond what framework improvements naturally provide)
- Integration with new testing tools not specified (Ginkgo/Gomega for Go, pytest for Python)
- Refactoring of production code to improve testability (unless minimal changes required for test migration)
- Changes to backend integration tests beyond framework conversion
- Migration of end-to-end (e2e) tests that may require different treatment

## Requirements

### Functional Requirements

#### Go Test Migration (Backend)
- **REQ-GO-1**: Convert all Go test files using standard `testing` package to Ginkgo+Gomega framework
- **REQ-GO-2**: Preserve all existing test logic and assertions during conversion
- **REQ-GO-3**: Update test suite structure to use Ginkgo's `Describe`, `Context`, `It`, `BeforeEach`, `AfterEach` blocks
- **REQ-GO-4**: Replace all `assert`/`require` calls with Gomega matchers (`Expect`, `Eventually`, `Consistently`)
- **REQ-GO-5**: Maintain test isolation and proper setup/teardown procedures
- **REQ-GO-6**: Support table-driven tests using Ginkgo's `DescribeTable` and `Entry` constructs

#### Python Test Migration (SDK)
- **REQ-PY-1**: Migrate all Python tests to use pytest framework exclusively
- **REQ-PY-2**: Move all test files from scattered locations to `sdk/tests/` directory structure
- **REQ-PY-3**: Organize tests by category: `sdk/tests/compilation/`, `sdk/tests/client/`, `sdk/tests/runtime/`, etc.
- **REQ-PY-4**: Convert test classes/functions to pytest conventions (test discovery, fixtures, parametrize)
- **REQ-PY-5**: Replace existing test runners/frameworks with pytest
- **REQ-PY-6**: Create conftest.py files for shared fixtures and configuration

#### Code Duplication Elimination
- **REQ-DUP-1**: Identify and consolidate duplicate test utility functions across Go tests
- **REQ-DUP-2**: Create shared test utilities package for Go: `backend/test/testutil/`
- **REQ-DUP-3**: Identify and consolidate duplicate test utilities across Python tests
- **REQ-DUP-4**: Create shared test utilities module for Python: `sdk/tests/test_utils/`
- **REQ-DUP-5**: Extract common test fixtures, mock data, and helper functions

### Non-Functional Requirements

#### Performance
- **REQ-PERF-1**: Test migration must not significantly increase overall test execution time (max 10% increase acceptable)
- **REQ-PERF-2**: Support parallel test execution capabilities (Ginkgo's `-p` flag, pytest-xdist)

#### Reliability
- **REQ-REL-1**: All migrated tests must pass with same or better reliability
- **REQ-REL-2**: Eliminate flaky tests identified during migration process
- **REQ-REL-3**: Maintain test isolation to prevent cascading failures

#### Maintainability
- **REQ-MAINT-1**: Establish consistent test naming conventions across all tests
- **REQ-MAINT-2**: Provide clear documentation for test structure and organization
- **REQ-MAINT-3**: Create style guide for writing new tests with Ginkgo+Gomega and pytest

#### Compatibility
- **REQ-COMPAT-1**: Maintain compatibility with existing CI/CD pipelines (minimal Makefile/script changes)
- **REQ-COMPAT-2**: Support both local development test execution and CI environments
- **REQ-COMPAT-3**: Maintain compatibility with existing test data and fixtures

#### KFP-Specific Requirements
- **REQ-KFP-1**: Maintain test coverage for all pipeline compilation scenarios
- **REQ-KFP-2**: Preserve integration test coverage for API endpoints and workflows
- **REQ-KFP-3**: Support testing of KFP metadata, artifacts, and pipeline versioning
- **REQ-KFP-4**: Ensure tests validate KFP component specifications and execution

### MVP vs. Nice-to-Have

#### MVP (Phase 1)
- Go backend integration tests migration (backend/test/integration/)
- Python SDK compilation tests migration (sdk/python/test/compilation/)
- Core test utilities consolidation
- Basic directory structure reorganization
- CI/CD pipeline updates

#### Nice-to-Have (Future Phases)
- Go unit tests migration for individual packages
- Python client library tests migration
- Test coverage reporting improvements
- Test execution performance optimization
- Advanced test fixtures and factories
- Mutation testing integration

## Done - Acceptance Criteria

### Migration Completeness
- [ ] All Go test files in `backend/test/` use Ginkgo+Gomega framework
- [ ] All Python test files are located in `sdk/tests/` directory structure
- [ ] All Python tests use pytest framework exclusively
- [ ] No duplicate test utility code exists across test files

### Test Execution
- [ ] All migrated tests pass in local development environment
- [ ] All migrated tests pass in CI/CD environment
- [ ] `make test` command executes all Go tests successfully
- [ ] `pytest sdk/tests/` executes all Python tests successfully
- [ ] Test execution time remains within acceptable limits (< 10% increase)

### Code Quality
- [ ] All test files follow established naming conventions
- [ ] Test code passes linting and formatting checks
- [ ] Common utilities are properly documented and reusable
- [ ] No deprecated testing patterns remain in codebase

### Documentation
- [ ] Test organization structure documented in README
- [ ] Style guide created for writing new tests
- [ ] Migration guide available for reference
- [ ] CI/CD configuration updated and documented

### Coverage
- [ ] Test coverage metrics maintained or improved
- [ ] No previously passing tests are now skipped or removed
- [ ] All critical pipeline execution paths remain tested

## Use Cases - User Experience & Workflow

### Use Case 1: Developer Running Tests Locally (Go Backend)

**Actor**: Backend Developer

**Workflow**:
1. Developer makes changes to backend Go code
2. Developer runs `make test-backend` or `ginkgo -r backend/test/integration/`
3. Ginkgo executes tests with descriptive BDD-style output
4. Test failures show clear, readable error messages with Gomega matchers
5. Developer can focus specific tests using `ginkgo --focus="pipeline API"` or similar
6. Developer can see hierarchical test structure in output (Describe > Context > It)
7. Tests provide actionable feedback for fixing issues

**Benefits**:
- Improved test output readability with BDD structure
- Better test filtering and focus capabilities
- Clearer failure messages with Gomega matchers
- Faster debugging with descriptive test names

### Use Case 2: Developer Running Tests Locally (Python SDK)

**Actor**: SDK Developer

**Workflow**:
1. Developer makes changes to Python SDK code
2. Developer runs `pytest sdk/tests/` or `pytest sdk/tests/compilation/`
3. Pytest discovers and executes tests with clear output
4. Developer can use markers: `pytest -m compilation` to run specific test categories
5. Test failures show clear assertion messages and diffs
6. Developer can run specific test file: `pytest sdk/tests/compilation/pipeline_compilation_test.py`
7. Developer can debug with `pytest --pdb` to drop into debugger on failure

**Benefits**:
- Unified test execution command across all Python tests
- Rich plugin ecosystem (coverage, xdist, etc.)
- Better test parametrization and fixtures
- Improved debugging capabilities

### Use Case 3: CI/CD Pipeline Execution

**Actor**: CI/CD System

**Workflow**:
1. Code changes trigger CI/CD pipeline
2. Pipeline runs `make test` or similar command
3. Go tests execute via Ginkgo with JUnit XML output for reporting
4. Python tests execute via pytest with coverage and JUnit XML output
5. Test results parsed and displayed in CI/CD dashboard
6. Failures trigger notifications with clear error messages
7. Coverage reports generated and tracked over time

**Benefits**:
- Consistent test execution interface
- Better test reporting in CI/CD dashboards
- Improved failure diagnostics
- Coverage tracking integration

### Use Case 4: Developer Writing New Tests

**Actor**: Any Developer

**Workflow**:
1. Developer needs to add tests for new feature
2. Developer reviews test style guide and examples
3. For Go: Developer creates new Ginkgo test with Describe/It blocks
4. For Python: Developer creates new pytest test with appropriate fixtures
5. Developer uses existing test utilities from testutil packages
6. Developer runs tests locally to verify
7. Tests pass code review with consistent style

**Benefits**:
- Clear patterns to follow
- Reusable utilities reduce boilerplate
- Consistent code review experience
- Easier onboarding for new contributors

### Use Case 5: Test Failure Investigation

**Actor**: Developer or QA Engineer

**Workflow**:
1. Test failure occurs in CI or locally
2. Test output shows hierarchical context (Ginkgo) or clear test name (pytest)
3. Assertion failure shows expected vs actual with clear formatting (Gomega/pytest)
4. Developer can re-run specific failing test in isolation
5. Developer can add additional debug output or use debugger
6. Root cause identified and fixed efficiently

**Benefits**:
- Faster time to identify root cause
- Better error messages reduce investigation time
- Easy test isolation for debugging
- Improved developer productivity

## Technical Architecture

### Go Test Architecture (Ginkgo + Gomega)

#### Current State
- Tests use standard Go `testing` package
- Assertions via `testify/assert` and `testify/require`
- Test suites via `testify/suite`
- Test utilities scattered across `backend/test/test_utils.go` and inline helpers

#### Target State
```
backend/
├── test/
│   ├── testutil/                    # Shared test utilities
│   │   ├── client_helpers.go        # API client helpers
│   │   ├── fixtures.go              # Test data fixtures
│   │   ├── matchers.go              # Custom Gomega matchers
│   │   └── suite.go                 # Common test suite setup
│   ├── integration/
│   │   ├── integration_suite_test.go    # Ginkgo suite entry point
│   │   ├── pipeline_api_test.go         # Pipeline API specs
│   │   ├── run_api_test.go              # Run API specs
│   │   └── ...
│   └── ...
```

#### Migration Pattern Example

**Before (testify)**:
```go
func TestPipelineAPI(t *testing.T) {
    suite.Run(t, new(PipelineApiTest))
}

func (s *PipelineApiTest) TestCreatePipeline() {
    pipeline, err := s.client.Create(params)
    require.Nil(s.T(), err)
    assert.Equal(s.T(), "test", pipeline.Name)
}
```

**After (Ginkgo + Gomega)**:
```go
var _ = Describe("Pipeline API", func() {
    var client *PipelineClient

    BeforeEach(func() {
        client = setupClient()
    })

    Context("when creating a pipeline", func() {
        It("should create pipeline successfully", func() {
            pipeline, err := client.Create(params)
            Expect(err).NotTo(HaveOccurred())
            Expect(pipeline.Name).To(Equal("test"))
        })
    })
})
```

#### Key Components
- **Test Suites**: Ginkgo suite files (`*_suite_test.go`) with `RunSpecs()`
- **Test Specs**: BDD-style test organization with `Describe`, `Context`, `It`
- **Setup/Teardown**: `BeforeEach`, `AfterEach`, `BeforeSuite`, `AfterSuite`
- **Assertions**: Gomega matchers (`Expect`, `Eventually`, `Consistently`)
- **Table Tests**: `DescribeTable` with `Entry` for parameterized tests

### Python Test Architecture (pytest)

#### Current State
- Tests scattered across multiple locations:
  - `sdk/python/test/`
  - `sdk/python/kfp/client/*_test.py`
  - `backend/src/apiserver/visualization/test_*.py`
- Mix of unittest and pytest patterns
- Test utilities in `sdk/python/test/test_utils/`

#### Target State
```
sdk/
├── tests/                           # All SDK tests consolidated here
│   ├── conftest.py                  # Shared pytest configuration
│   ├── test_utils/                  # Shared test utilities
│   │   ├── __init__.py
│   │   ├── comparison_utils.py
│   │   ├── file_utils.py
│   │   └── fixtures.py              # pytest fixtures
│   ├── compilation/
│   │   ├── __init__.py
│   │   ├── conftest.py              # Compilation-specific fixtures
│   │   └── test_pipeline_compilation.py
│   ├── client/
│   │   ├── __init__.py
│   │   ├── conftest.py
│   │   ├── test_client.py
│   │   └── test_auth.py
│   ├── runtime/
│   │   └── test_execute_commands.py
│   └── local_execution/
│       └── test_local_execution.py
```

#### Migration Pattern Example

**Before**:
```python
class TestPipelineCompilation:
    def test_compilation(self, pipeline_data):
        temp_file = os.path.join(tempfile.gettempdir(), pipeline_data.compiled_file_name)
        try:
            Compiler().compile(...)
            # assertions
        finally:
            os.remove(temp_file)
```

**After**:
```python
@pytest.fixture
def temp_output_file(tmp_path):
    """Provides temporary file for compilation output."""
    output_file = tmp_path / "pipeline.yaml"
    yield output_file
    # Automatic cleanup by pytest tmp_path

def test_pipeline_compilation(pipeline_data, temp_output_file):
    """Test pipeline compilation produces expected output."""
    Compiler().compile(
        pipeline_func=pipeline_data.pipeline_func,
        package_path=temp_output_file,
    )
    # assertions using pytest's rich assertion rewriting
    assert temp_output_file.exists()
```

#### Key Components
- **Test Discovery**: Automatic via `test_*.py` or `*_test.py` naming
- **Fixtures**: Reusable test setup via `@pytest.fixture` decorator
- **Parametrization**: `@pytest.mark.parametrize` for data-driven tests
- **Markers**: Custom markers for test categorization (`@pytest.mark.compilation`)
- **Plugins**: pytest-cov for coverage, pytest-xdist for parallel execution

### Test Utilities Consolidation

#### Go Test Utilities (`backend/test/testutil/`)
- **client_helpers.go**: API client creation, authentication helpers
- **fixtures.go**: Common test data, pipeline definitions, resource fixtures
- **matchers.go**: Custom Gomega matchers for KFP-specific assertions
- **suite.go**: Common suite setup, namespace management, cleanup utilities
- **constants.go**: Shared test constants, file paths, configuration

#### Python Test Utilities (`sdk/tests/test_utils/`)
- **comparison_utils.py**: Pipeline spec comparison, YAML diff utilities
- **file_utils.py**: File I/O helpers, test data loading
- **fixtures.py**: Pytest fixtures for common test resources
- **builders.py**: Test data builders for pipelines, components, parameters
- **matchers.py**: Custom assertion helpers for complex objects

### CI/CD Integration

#### Makefile Changes
```makefile
# Go tests
.PHONY: test-backend
test-backend:
    ginkgo -r -p --randomize-all --cover backend/test/integration/

# Python tests
.PHONY: test-sdk
test-sdk:
    pytest sdk/tests/ -v --cov=sdk/python/kfp --cov-report=html

# All tests
.PHONY: test
test: test-backend test-sdk
```

#### GitHub Actions Integration
- Install Ginkgo CLI in CI environment
- Configure pytest with appropriate plugins
- Generate JUnit XML reports for both frameworks
- Collect and publish coverage reports
- Cache test dependencies for faster execution

### Data Flow and Dependencies

1. **Test Execution Flow**:
   - Developer/CI triggers test command
   - Test framework discovers and loads tests
   - Setup/fixtures execute before tests
   - Tests execute with assertions
   - Teardown/cleanup executes after tests
   - Results aggregated and reported

2. **Dependencies**:
   - Go: Ginkgo v2.x, Gomega v1.x
   - Python: pytest v7.x+, pytest-cov, pytest-xdist
   - Test data files in `backend/test/resources/` and `sdk/python/test_data/`
   - API server and database for integration tests

### Resource Requirements and Constraints

- **Development Environment**:
  - Go 1.19+ with Ginkgo/Gomega installed
  - Python 3.8+ with pytest installed
  - Existing KFP test infrastructure (databases, API servers)

- **CI Environment**:
  - Additional dependencies in CI container images
  - Sufficient resources for parallel test execution
  - Test result storage and reporting infrastructure

### Security and Compliance Considerations

- **Test Data Security**: Ensure no sensitive data in test fixtures
- **Secret Management**: Use secure methods for test credentials/tokens
- **Access Control**: Tests should not require elevated permissions
- **Audit Trail**: Test execution logs for compliance requirements

## Documentation Considerations

### User Documentation Updates

1. **Developer Guide - Testing Section**:
   - How to run tests locally (Go and Python)
   - Test organization and structure
   - Writing new tests following established patterns
   - Debugging test failures

2. **Contributing Guide**:
   - Test requirements for pull requests
   - Test style guidelines and conventions
   - How to add test coverage for new features

3. **Test Organization README** (`backend/test/README.md` and `sdk/tests/README.md`):
   - Directory structure explanation
   - Purpose of each test category
   - How to run specific test suites
   - Common test utilities and their usage

### API Documentation Requirements

- No direct API documentation changes required
- Test utilities should have inline documentation (godoc/docstrings)

### Examples and Tutorials to Create

1. **Ginkgo Test Writing Tutorial**:
   - Basic test structure
   - Using Describe/Context/It effectively
   - Common Gomega matchers for KFP
   - Table-driven tests with DescribeTable

2. **Pytest Test Writing Tutorial**:
   - Creating test functions and classes
   - Using fixtures effectively
   - Parametrization patterns
   - Custom markers and configuration

3. **Test Utilities Guide**:
   - Available helper functions
   - Creating reusable fixtures
   - Best practices for test data management

### Migration Guide

- **Step-by-step migration process** for developers updating tests
- **Before/after examples** for common test patterns
- **Troubleshooting common migration issues**
- **Timeline and phased rollout plan**

## Questions to Answer

### Technical Decisions

1. **Q**: Should we migrate all tests at once or in phases?
   - **A**: Phased approach recommended - start with integration tests, then unit tests

2. **Q**: How do we handle tests that are currently flaky or failing?
   - **A**: Use migration as opportunity to fix or mark with pending/skip appropriately

3. **Q**: Should we maintain backward compatibility during migration (support both frameworks)?
   - **A**: No - clean cutover preferred to avoid maintenance burden of dual frameworks

4. **Q**: What custom Gomega matchers should we create for KFP-specific assertions?
   - **A**: Matchers for pipeline specs, component definitions, API responses, resource states

5. **Q**: Should we use Ginkgo v1 or v2?
   - **A**: Ginkgo v2 - latest stable with better features and performance

### Integration Concerns

1. **Q**: How do we ensure CI/CD pipelines continue working during migration?
   - **A**: Update CI configurations as part of migration PRs, test thoroughly in feature branches

2. **Q**: What happens to test history and trending data?
   - **A**: Maintain test names/identifiers where possible; document any breaking changes

3. **Q**: How do we handle tests that span multiple services (e2e tests)?
   - **A**: Migrate to Ginkgo for Go-based e2e tests; maintain separate strategy for multi-language tests

4. **Q**: Should we change test execution order or parallelization strategy?
   - **A**: Leverage Ginkgo/pytest built-in parallelization; ensure tests remain independent

### Testing Strategy

1. **Q**: How do we validate that migrated tests have equivalent coverage?
   - **A**: Compare coverage reports before/after; manual review of test logic during migration

2. **Q**: Should we add new tests during migration or keep scope limited?
   - **A**: Limit scope to migration; new tests should be separate effort

3. **Q**: How do we handle deprecated or obsolete tests?
   - **A**: Use migration as opportunity to remove outdated tests with proper justification

4. **Q**: What's the rollback plan if migration causes major issues?
   - **A**: Maintain migration in feature branch; thorough testing before merge; git revert as fallback

### Team Coordination

1. **Q**: How do we coordinate migration across multiple developers/teams?
   - **A**: Assign ownership by test directory; regular sync meetings; shared tracking document

2. **Q**: What training or resources do developers need?
   - **A**: Provide Ginkgo/pytest tutorials, example PRs, office hours for questions

3. **Q**: How do we handle merge conflicts during extended migration period?
   - **A**: Migrate in logical chunks; communicate clearly about affected areas; prioritize quick merges

4. **Q**: Who reviews test migration PRs?
   - **A**: Require review from test infrastructure owners plus original test authors where possible
