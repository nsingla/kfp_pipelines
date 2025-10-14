---
description: Generate detailed feature specifications for KFP pipeline development based on requirements.
---

The user input to you can be provided directly by the agent or as a command argument - you **MUST** consider it before proceeding with the prompt (if not empty).

User input:

$ARGUMENTS

Given the feature requirements provided as an argument, do this:

1. **Analyze Requirements**: Parse the input to understand:
   - Core feature objectives for KFP pipelines
   - User personas and use cases
   - Technical constraints and dependencies
   - Performance and scalability requirements
   - Integration points with existing KFP infrastructure

2. **Generate Feature Specification**: Create a comprehensive spec.md that includes:
   
   ## Feature Title
   Clear, descriptive name for the KFP pipeline feature
   
   ## Feature Overview
   Executive summary describing the pipeline capability, value proposition, and user impact
   
   ## Goals
   - Primary objectives for pipeline users
   - Expected outcomes and benefits
   - Success metrics specific to KFP workflows
   
   ## Out of Scope
   - Features explicitly not included
   - Future considerations
   - Alternative approaches not pursued
   
   ## Requirements
   - Functional requirements for pipeline components
   - Non-functional requirements (performance, reliability, etc.)
   - MVP vs. nice-to-have features
   - KFP-specific requirements (versioning, metadata, etc.)
   
   ## Done - Acceptance Criteria
   - Testable conditions for feature completion
   - Pipeline execution success criteria
   - User experience validation points
   
   ## Use Cases - User Experience & Workflow
   - Step-by-step user workflows
   - Pipeline execution scenarios
   - Integration with KFP UI and SDK
   - Error handling and recovery scenarios
   
   ## Technical Architecture
   - Pipeline component design
   - Data flow and dependencies
   - Resource requirements and constraints
   - Security and compliance considerations
   
   ## Documentation Considerations
   - User documentation updates needed
   - API documentation requirements
   - Examples and tutorials to create
   
   ## Questions to Answer
   - Technical decisions requiring clarification
   - Integration concerns
   - Testing strategy questions

3. **Validate Specification**: Ensure the spec addresses:
   - KFP best practices and patterns
   - Compatibility with existing pipeline infrastructure
   - Scalability and performance considerations
   - Security and access control requirements

4. **Output**: Generate the specification as specs/spec.md in the current feature directory.

Use KFP-specific terminology and patterns throughout the specification.
