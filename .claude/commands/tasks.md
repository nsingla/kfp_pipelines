---
description: Generate detailed implementation tasks for KFP pipeline features based on specifications and plans.
---

The user input to you can be provided directly by the agent or as a command argument - you **MUST** consider it before proceeding with the prompt (if not empty).

User input:

$ARGUMENTS

Given the feature specification and implementation plan, generate detailed tasks for KFP pipeline development:

1. **Load Prerequisites**: Read and analyze:
   - Feature specification (spec.md)
   - Implementation plan (plan.md)
   - Any existing pipeline components or infrastructure

2. **Break Down Implementation**: Create detailed, actionable tasks covering:

   ## Development Tasks
   - Pipeline component development
   - SDK integration and API updates
   - Configuration and metadata handling
   - Resource management implementation

   ## Testing Tasks
   - Unit tests for pipeline components
   - Integration tests with KFP infrastructure
   - End-to-end pipeline execution tests
   - Performance and scalability testing
   - Security and access control validation

   ## Documentation Tasks
   - API documentation updates
   - User guide creation/updates
   - Code examples and tutorials
   - Migration guides (if applicable)

   ## Infrastructure Tasks
   - Deployment configuration updates
   - CI/CD pipeline modifications
   - Monitoring and observability setup
   - Resource allocation and scaling configuration

   ## Validation Tasks
   - Acceptance criteria verification
   - User experience validation
   - Performance benchmark validation
   - Compatibility testing across KFP versions

3. **Task Structure**: For each task, provide:
   - **Task ID**: Unique identifier
   - **Title**: Clear, actionable description
   - **Description**: Detailed requirements and context
   - **Prerequisites**: Dependencies on other tasks
   - **Acceptance Criteria**: How to verify completion
   - **Estimated Effort**: Time/complexity estimate
   - **Owner/Skills**: Required expertise
   - **KFP Components**: Affected pipeline components

4. **Prioritization**: Order tasks by:
   - Critical path dependencies
   - Risk and complexity
   - Value delivery sequence
   - Resource availability

5. **Validation**: Ensure tasks cover:
   - All specification requirements
   - KFP best practices and patterns
   - Testing and quality assurance
   - Documentation and user experience

Output the tasks as a structured tasks.md file with clear sections and actionable items.
