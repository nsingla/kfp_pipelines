# ML Pipeline Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]
**Input**: ML feature specification from `/specs/[###-feature-name]/spec.md`

## Execution Flow (/plan command scope for ML)
```
1. Load ML feature spec from Input path
   → If not found: ERROR "No ML feature spec at {path}"
2. Fill ML Technical Context (scan for NEEDS CLARIFICATION)
   → Detect ML Project Type (pipeline components, SDK extensions, API enhancements)
   → Set ML Architecture Decision based on KFP patterns
3. Fill the Constitution Check section based on ML platform constitution
4. Evaluate Constitution Check section for ML requirements
   → If violations exist: Document in ML Complexity Tracking
   → If no justification possible: ERROR "Simplify ML approach first"
   → Update Progress Tracking: Initial ML Constitution Check
5. Execute Phase 0 → ml-research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve ML unknowns"
6. Execute Phase 1 → ml-contracts, ml-data-model.md, ml-quickstart.md, CLAUDE.md
7. Re-evaluate Constitution Check section for ML platform
   → If new violations: Refactor ML design, return to Phase 1
   → Update Progress Tracking: Post-Design ML Constitution Check
8. Plan Phase 2 → Describe ML task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 8. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: ML implementation execution

## Summary
[Extract from ML feature spec: primary ML requirement + technical approach from research]

## ML Technical Context
**ML Framework/Version**: [e.g., TensorFlow 2.15, PyTorch 2.1, XGBoost 2.0 or NEEDS CLARIFICATION]
**KFP SDK Version**: [e.g., kfp>=2.0.0 or NEEDS CLARIFICATION]
**Primary ML Dependencies**: [e.g., scikit-learn, pandas, numpy or NEEDS CLARIFICATION]
**Container Runtime**: [e.g., Docker, Podman, containerd or NEEDS CLARIFICATION]
**Storage Backend**: [e.g., MinIO, GCS, S3, local PVC or NEEDS CLARIFICATION]
**Resource Requirements**: [e.g., GPU support, memory limits, CPU requirements or NEEDS CLARIFICATION]
**Target Kubernetes**: [e.g., vanilla k8s 1.28+, OpenShift 4.12+ or NEEDS CLARIFICATION]
**ML Project Type**: [pipeline-component/sdk-extension/api-enhancement - determines implementation structure]
**Performance Goals**: [ML-specific, e.g., pipeline execution <30min, model inference <100ms or NEEDS CLARIFICATION]
**ML Constraints**: [e.g., model size <1GB, training time <4hrs, multi-tenant isolation or NEEDS CLARIFICATION]
**Scale/Scope**: [ML-specific, e.g., 100 concurrent pipelines, 10TB datasets, multi-framework support or NEEDS CLARIFICATION]

## Constitution Check for ML Platform
*GATE: Must pass before Phase 0 ML research. Re-check after Phase 1 ML design.*

[Gates determined based on ML platform constitution file]

## ML Project Structure

### Documentation (this ML feature)
```
specs/[###-ml-feature]/
├── plan.md              # This file (/plan command output)
├── ml-research.md       # Phase 0 output (/plan command)
├── ml-data-model.md     # Phase 1 output (/plan command)
├── ml-quickstart.md     # Phase 1 output (/plan command)
├── ml-contracts/        # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### ML Source Code (repository root)
```
# Option 1: Pipeline Component (DEFAULT for ML components)
components/
├── [component-name]/
│   ├── component.py
│   ├── Dockerfile
│   └── component.yaml
└── tests/
    ├── unit/
    ├── integration/
    └── e2e/

# Option 2: KFP SDK Extension (when extending SDK functionality)
kfp_extensions/
├── src/
│   ├── pipeline_ops/
│   ├── component_utils/
│   └── metadata_extensions/
└── tests/
    ├── unit/
    ├── integration/
    └── sdk_compatibility/

# Option 3: ML Platform API Enhancement (when extending KFP API server)
backend/
├── src/
│   ├── api/
│   ├── models/
│   ├── services/
│   └── ml_handlers/
└── tests/
    ├── unit/
    ├── contract/
    └── integration/

# Option 4: Full ML Pipeline Implementation
pipelines/
├── [pipeline-name]/
│   ├── pipeline.py
│   ├── components/
│   ├── configs/
│   └── tests/
└── shared/
    ├── utils/
    ├── base_components/
    └── ml_frameworks/
```

**ML Architecture Decision**: [Document the selected ML structure and reference the real directories]

## Phase 0: ML Outline & Research
1. **Extract ML unknowns from Technical Context** above:
   - For each NEEDS CLARIFICATION → ML research task
   - For each ML framework → compatibility and best practices task
   - For each KFP integration → patterns and API task

2. **Generate and dispatch ML research agents**:
   ```
   For each ML unknown in Technical Context:
     Task: "Research {ML framework/tool} for {KFP feature context}"
   For each ML technology choice:
     Task: "Find best practices for {ML tech} in {KFP domain}"
   For each KFP integration:
     Task: "Research KFP patterns for {integration type}"
   ```

3. **Consolidate ML findings** in `ml-research.md` using format:
   - ML Decision: [what ML approach was chosen]
   - ML Rationale: [why chosen for KFP context]
   - ML Alternatives considered: [what other ML approaches evaluated]
   - KFP Integration: [how it fits with existing KFP patterns]

**Output**: ml-research.md with all ML NEEDS CLARIFICATION resolved

## Phase 1: ML Design & Contracts
*Prerequisites: ml-research.md complete*

1. **Extract ML entities from feature spec** → `ml-data-model.md`:
   - Pipeline component specifications (inputs, outputs, parameters)
   - ML artifact schemas (models, datasets, metrics)
   - Metadata and lineage requirements
   - Resource specifications and constraints

2. **Generate ML API contracts** from functional requirements:
   - For each ML operation → component interface or API endpoint
   - Use KFP component specification patterns
   - Output component YAML specs to `/ml-contracts/`
   - Include parameter validation and type checking

3. **Generate ML contract tests** from contracts:
   - One test file per ML component or API endpoint
   - Assert component input/output schemas
   - Validate ML framework integration
   - Tests must fail (no implementation yet)

4. **Extract ML test scenarios** from user stories:
   - Each ML story → end-to-end pipeline test scenario
   - ML quickstart test = complete pipeline execution validation
   - Include performance and resource validation

5. **Update CLAUDE.md incrementally** for ML context:
   - Add NEW ML technologies from current plan
   - Include KFP-specific patterns and constraints
   - Update ML framework compatibility notes
   - Keep ML pipeline context under 150 lines

**Output**: ml-data-model.md, /ml-contracts/*, failing ML tests, ml-quickstart.md, CLAUDE.md

## Phase 2: ML Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**ML Task Generation Strategy**:
- Load `rfe/templates/tasks-template.md` as base
- Generate ML tasks from Phase 1 design docs
- Each ML contract → component test task [P]
- Each ML entity → component/model creation task [P]
- Each ML story → end-to-end pipeline test task
- ML framework integration tasks to make tests pass

**ML Ordering Strategy**:
- TDD order: ML tests before implementation
- Dependency order: Base components before composite pipelines
- Resource order: Local tests before GPU/distributed tests
- Mark [P] for parallel execution (independent ML components)

**Estimated Output**: 25-35 numbered, ordered ML tasks in tasks.md

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future ML Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: ML task execution (/tasks command creates tasks.md)
**Phase 4**: ML implementation (execute ML tasks.md following KFP patterns)
**Phase 5**: ML validation (run pipeline tests, execute ml-quickstart.md, performance validation)

## ML Complexity Tracking
*Fill ONLY if Constitution Check has ML violations that must be justified*

| ML Violation | Why Needed | Simpler ML Alternative Rejected Because |
|--------------|------------|------------------------------------------|
| [e.g., Custom ML framework] | [specific ML need] | [why standard frameworks insufficient] |
| [e.g., GPU requirements] | [performance requirement] | [why CPU-only insufficient] |

## Progress Tracking
*This checklist is updated during ML execution flow*

**ML Phase Status**:
- [ ] Phase 0: ML Research complete (/plan command)
- [ ] Phase 1: ML Design complete (/plan command)
- [ ] Phase 2: ML Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: ML Tasks generated (/tasks command)
- [ ] Phase 4: ML Implementation complete
- [ ] Phase 5: ML Validation passed

**ML Gate Status**:
- [ ] Initial ML Constitution Check: PASS
- [ ] Post-Design ML Constitution Check: PASS
- [ ] All ML NEEDS CLARIFICATION resolved
- [ ] ML complexity deviations documented
- [ ] KFP integration patterns validated

---
*Based on KFP Constitution - See `rfe/memory/constitution.md`*