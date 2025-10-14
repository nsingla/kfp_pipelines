---
description: Generate an RFE (Request for Enhancement) document for KFP pipeline features.
---

The user input to you can be provided directly by the agent or as a command argument - you **MUST** consider it before proceeding with the prompt (if not empty).

User input:

$ARGUMENTS

Create an RFE (Request for Enhancement) document based on the provided requirements. Generate an rfe.md file with the following structure:

# Feature Title

**Feature Overview:**  
*An elevator pitch (value statement) that describes the KFP Pipeline Feature in a clear, concise way. Executive Summary of the user goal or problem that is being solved, why does this matter to pipeline developers and users? The "What & Why"...*

**Goals:**

*Provide high-level goal statement, providing user context and expected user outcome(s) for this KFP Pipeline Feature. Who benefits from this Feature (data scientists, ML engineers, DevOps teams), and how? What is the difference between today's current state and a world with this Feature in the KFP ecosystem?*

**Out of Scope:**

*High-level list of items, features, or personas that are out of scope for this KFP pipeline enhancement.*

**Requirements:**

*A list of specific needs, capabilities, or objectives that the KFP Pipeline Feature must deliver to satisfy the Feature. Some requirements will be flagged as MVP. If an MVP gets shifted, the Feature shifts. If a non MVP requirement slips, it does not shift the feature. Consider KFP-specific requirements like:*
- Pipeline component compatibility
- Kubeflow integration requirements
- SDK compatibility
- Metadata and artifact management
- Resource management and scaling

**Done - Acceptance Criteria:**

*Acceptance Criteria articulates and defines the value proposition - what is required to meet the goal and intent of this KFP Pipeline Feature. The Acceptance Criteria provides a detailed definition of scope and the expected outcomes - from a pipeline user's point of view. Include criteria for:*
- Pipeline execution success
- Component functionality
- Integration testing
- Performance benchmarks

**Use Cases - User Experience & Workflow:**

*Include use case diagrams, main success scenarios, alternative flow scenarios specific to KFP pipeline workflows. Consider different user personas:*
- Data Scientists creating experiments
- ML Engineers building production pipelines
- DevOps teams managing pipeline infrastructure

**Documentation Considerations:**

*Provide information that needs to be considered and planned so that documentation will meet KFP community needs. If the feature extends existing KFP functionality, provide a link to its current documentation.*

**Questions to Answer:**

*Include a list of refinement / architectural questions that may need to be answered before coding can begin. Consider KFP-specific concerns:*
- Component versioning strategy
- Backward compatibility requirements
- Integration with KFP UI
- Resource allocation and limits
- Security and access control

**Background & Strategic Fit:**

*Provide any additional context needed to frame the feature within the broader KFP ecosystem and roadmap.*

**Customer Considerations**

*Provide any additional customer-specific considerations that must be made when designing and delivering the KFP Pipeline Feature. Consider different deployment scenarios and user environments.*

Focus on KFP-specific terminology, patterns, and use cases throughout the document.
