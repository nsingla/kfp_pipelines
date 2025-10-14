---
description: Execute implementation tasks for KFP pipeline features with code generation and testing.
---

The user input to you can be provided directly by the agent or as a command argument - you **MUST** consider it before proceeding with the prompt (if not empty).

User input:

$ARGUMENTS

Execute the implementation phase for KFP pipeline features:

1. **Load Implementation Context**: Read and analyze:
   - Feature specification (spec.md)
   - Implementation plan (plan.md)
   - Task breakdown (tasks.md)
   - Existing codebase and pipeline infrastructure

2. **Implementation Execution**: For each task in priority order:

   ## Code Development
   - Generate KFP pipeline components using best practices
   - Implement SDK integrations and API endpoints
   - Create configuration management and metadata handling
   - Develop resource management and scaling logic
   - Follow KFP coding standards and patterns

   ## Testing Implementation
   - Create comprehensive unit tests for all components
   - Implement integration tests with KFP infrastructure
   - Build end-to-end pipeline execution tests
   - Add performance and load testing scenarios
   - Validate security and access control mechanisms

   ## Documentation Generation
   - Update API documentation with new endpoints
   - Create user guides and tutorials
   - Generate code examples and sample pipelines
   - Write migration guides for breaking changes

   ## Infrastructure Updates
   - Update deployment configurations
   - Modify CI/CD pipelines for new components
   - Set up monitoring and alerting
   - Configure resource limits and scaling policies

3. **Quality Assurance**: For each implementation:
   - Validate against acceptance criteria
   - Run comprehensive test suites
   - Perform code reviews and security analysis
   - Verify KFP integration and compatibility
   - Test pipeline execution in various scenarios

4. **Delivery Preparation**:
   - Package components for deployment
   - Prepare release notes and changelogs
   - Create deployment runbooks
   - Set up rollback procedures
   - Validate in staging environment

5. **Progress Tracking**: Update task status and provide:
   - Implementation completion status
   - Test results and coverage metrics
   - Performance benchmarks
   - Known issues and limitations
   - Deployment readiness assessment

Focus on KFP-specific patterns, component lifecycle, and pipeline execution best practices throughout implementation.
