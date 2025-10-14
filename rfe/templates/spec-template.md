# ML Pipeline Feature Specification: [FEATURE NAME]

**Feature Branch**: `[###-feature-name]`
**Created**: [DATE]
**Status**: Draft
**Input**: ML feature description: "$ARGUMENTS"

## Execution Flow (main)
```
1. Parse ML feature description from Input
   → If empty: ERROR "No ML pipeline feature description provided"
2. Extract key ML concepts from description
   → Identify: ML workflows, pipeline components, data artifacts, performance constraints
3. For each unclear ML aspect:
   → Mark with [NEEDS CLARIFICATION: specific ML question]
4. Fill ML User Scenarios & Testing section
   → If no clear ML workflow: ERROR "Cannot determine ML pipeline scenarios"
5. Generate ML Functional Requirements
   → Each requirement must be testable in KFP environment
   → Mark ambiguous ML requirements
6. Identify Key ML Entities (pipeline components, artifacts, metadata)
7. Run ML Review Checklist
   → If any [NEEDS CLARIFICATION]: WARN "ML spec has uncertainties"
   → If implementation details found: ERROR "Remove tech details"
8. Return: SUCCESS (ML spec ready for planning)
```

---

## ⚡ Quick Guidelines for ML Pipeline Features
- ✅ Focus on WHAT ML users need and WHY
- ❌ Avoid HOW to implement (no specific ML frameworks, APIs, code structure)
- 👥 Written for ML practitioners and platform stakeholders, not developers
- 🤖 Consider data scientists, ML engineers, and platform operators as primary users

### Section Requirements for ML Features
- **Mandatory sections**: Must be completed for every ML pipeline feature
- **Optional sections**: Include only when relevant to the ML feature
- When a section doesn't apply, remove it entirely (don't leave as "N/A")

### For AI Generation of ML Specs
When creating this ML spec from a user prompt:
1. **Mark all ML ambiguities**: Use [NEEDS CLARIFICATION: specific ML question] for any assumption
2. **Don't guess ML specifics**: If the prompt doesn't specify (e.g., "model training" without framework), mark it
3. **Think like an ML tester**: Every vague requirement should fail the "testable in KFP environment" checklist
4. **Common underspecified ML areas**:
   - ML framework compatibility requirements
   - Resource requirements (CPU, memory, GPU)
   - Pipeline scalability and performance targets
   - Model versioning and artifact management
   - Integration with existing ML infrastructure
   - Security/compliance needs for ML data and models

---

## ML User Scenarios & Testing *(mandatory)*

### Primary ML User Story
[Describe the main ML workflow journey in plain language - data scientist to model deployment]

### ML Acceptance Scenarios
1. **Given** [ML pipeline state], **When** [ML action], **Then** [expected ML outcome]
2. **Given** [model training scenario], **When** [pipeline execution], **Then** [expected artifacts and metadata]
3. **Given** [ML infrastructure state], **When** [resource constraints], **Then** [expected scaling behavior]

### ML Edge Cases
- What happens when [ML pipeline component fails]?
- How does system handle [large dataset or model size]?
- What occurs during [resource exhaustion or GPU unavailability]?
- How does pipeline behave with [incompatible ML framework versions]?

## ML Pipeline Requirements *(mandatory)*

### Functional Requirements for ML Workflows
- **FR-001**: Pipeline MUST [specific ML capability, e.g., "support TensorFlow model training components"]
- **FR-002**: System MUST [ML data handling, e.g., "validate input data schemas for pipeline components"]
- **FR-003**: ML users MUST be able to [key ML interaction, e.g., "monitor pipeline execution progress and metrics"]
- **FR-004**: Pipeline MUST [ML artifact requirement, e.g., "persist model artifacts with versioning metadata"]
- **FR-005**: System MUST [ML behavior, e.g., "log all pipeline execution events and resource usage"]

*Example of marking unclear ML requirements:*
- **FR-006**: Pipeline MUST support [NEEDS CLARIFICATION: ML frameworks not specified - TensorFlow, PyTorch, XGBoost, scikit-learn?]
- **FR-007**: System MUST retain ML artifacts for [NEEDS CLARIFICATION: retention period for models and datasets not specified]
- **FR-008**: Pipeline MUST scale to [NEEDS CLARIFICATION: concurrent execution limits and resource requirements not defined]

### Non-Functional Requirements for ML Platform
- **NFR-001**: Pipeline execution MUST complete within [NEEDS CLARIFICATION: acceptable latency not specified]
- **NFR-002**: System MUST support [NEEDS CLARIFICATION: concurrent pipeline limit not specified] simultaneous executions
- **NFR-003**: ML artifacts MUST be accessible across [NEEDS CLARIFICATION: multi-tenancy requirements not defined]
- **NFR-004**: Pipeline MUST integrate with existing [NEEDS CLARIFICATION: required integrations not specified]

### Key ML Entities *(include if feature involves ML data/components)*
- **Pipeline Component**: [What ML operation it performs, input/output specifications]
- **ML Artifact**: [What model or data it represents, versioning and lineage requirements]
- **Execution Metadata**: [What pipeline information it captures, relationships to components]
- **Resource Specification**: [What computational requirements it defines, scaling parameters]

---

## ML Platform Integration *(include if applicable)*

### KFP Integration Points
- **SDK Integration**: [How feature integrates with KFP Python SDK]
- **API Server Integration**: [Required API changes or extensions]
- **Metadata Store**: [How feature uses or extends ML metadata]
- **UI Integration**: [Pipeline authoring or monitoring UI changes]

### External ML Dependencies
- **ML Frameworks**: [Required ML library versions and compatibility]
- **Container Runtime**: [Special container or image requirements]
- **Resource Management**: [GPU, storage, or networking requirements]
- **Data Storage**: [Integration with data lakes, databases, or artifact stores]

---

## Review & Acceptance Checklist for ML Features
*GATE: Automated checks run during main() execution*

### ML Content Quality
- [ ] No implementation details (specific ML frameworks, APIs, deployment mechanisms)
- [ ] Focused on ML user value and business needs
- [ ] Written for ML practitioners and platform stakeholders
- [ ] All mandatory ML sections completed
- [ ] Considers data scientist, ML engineer, and platform operator personas

### ML Requirement Completeness
- [ ] No [NEEDS CLARIFICATION] markers remain
- [ ] ML requirements are testable in KFP environment
- [ ] ML success criteria are measurable (accuracy, latency, throughput)
- [ ] ML pipeline scope is clearly bounded
- [ ] ML framework dependencies and assumptions identified
- [ ] Resource requirements and scaling constraints specified
- [ ] Security and compliance requirements for ML data addressed

---

## Execution Status
*Updated by main() during ML spec processing*

- [ ] ML feature description parsed
- [ ] Key ML concepts extracted
- [ ] ML ambiguities marked
- [ ] ML user scenarios defined
- [ ] ML pipeline requirements generated
- [ ] ML entities identified
- [ ] ML review checklist passed

---