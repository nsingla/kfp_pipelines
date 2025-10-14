---
description: Identify underspecified areas in the current KFP feature spec by asking up to 5 highly targeted clarification questions and encoding answers back into the spec.
---

The user input to you can be provided directly by the agent or as a command argument - you **MUST** consider it before proceeding with the prompt (if not empty).

User input:

$ARGUMENTS

Goal: Detect and reduce ambiguity or missing decision points in the active KFP feature specification and record the clarifications directly in the spec file.

Note: This clarification workflow is expected to run (and be completed) BEFORE invoking `/plan`. If the user explicitly states they are skipping clarification (e.g., exploratory spike), you may proceed, but must warn that downstream rework risk increases.

Execution steps:

1. Run `rfe/scripts/check-prerequisites.sh --json --paths-only` from repo root **once** (combined `--json --paths-only` mode). Parse minimal JSON payload fields:
   - `FEATURE_DIR`
   - `FEATURE_SPEC`
   - (Optionally capture `IMPL_PLAN`, `TASKS` for future chained flows.)
   - If JSON parsing fails, abort and instruct user to re-run `/specify` or verify feature branch environment.

2. Load the current spec file. Perform a structured ambiguity & coverage scan using this taxonomy for ML pipeline features. For each category, mark status: Clear / Partial / Missing. Produce an internal coverage map used for prioritization.

   ML Pipeline Functional Scope & Behavior:
   - Core ML workflow goals & pipeline success criteria
   - Explicit out-of-scope declarations for ML features
   - User roles / personas differentiation (data scientists, ML engineers, platform operators)

   ML Data Model & Pipeline Structure:
   - Pipeline components, inputs, outputs, and artifacts
   - Parameter schemas and validation rules
   - Component dependency relationships and execution order
   - Data lineage and artifact management
   - ML model versioning and metadata requirements

   ML Workflow & UX Flow:
   - Critical ML pipeline authoring journeys / sequences
   - Pipeline execution, monitoring, and debugging workflows
   - Error handling for failed pipeline steps
   - Integration with Jupyter notebooks and ML frameworks

   Non-Functional Quality Attributes for ML:
   - Performance (pipeline execution latency, throughput targets)
   - Scalability (concurrent pipeline execution, resource scaling)
   - Reliability & availability (pipeline failure recovery, retry strategies)
   - Observability (pipeline logging, metrics, monitoring)
   - Security & privacy (multi-tenant isolation, model security, data privacy)
   - Compliance / regulatory constraints for ML workflows

   ML Platform Integration & Dependencies:
   - Integration with KFP SDK, API server, and metadata store
   - Argo Workflows integration patterns and failure modes
   - ML framework compatibility (TensorFlow, PyTorch, XGBoost, scikit-learn)
   - Kubernetes resource management and GPU support
   - Container image management and artifact storage

   ML Pipeline Edge Cases & Failure Handling:
   - Pipeline component failure and retry scenarios
   - Resource exhaustion and quota management
   - Concurrent pipeline execution conflicts
   - Model serving integration failure modes

   ML Platform Constraints & Tradeoffs:
   - Technical constraints (Kubernetes versions, resource limits)
   - ML framework version compatibility
   - Explicit tradeoffs or rejected ML workflow alternatives

   ML Terminology & Consistency:
   - Canonical ML pipeline and component terms
   - Avoided synonyms / deprecated ML terminology

   ML Pipeline Completion Signals:
   - Pipeline execution acceptance criteria testability
   - Measurable Definition of Done for ML workflows

   For each category with Partial or Missing status, add a candidate question opportunity unless clarification would not materially change ML implementation or validation strategy.

3. Generate (internally) a prioritized queue of candidate clarification questions (maximum 5). Apply these constraints:
    - Maximum of 5 total questions across the whole session.
    - Each question must be answerable with EITHER:
       * A short multiple‑choice selection (2–5 distinct, mutually exclusive options), OR
       * A one-word / short‑phrase answer (explicitly constrain: "Answer in <=5 words").
   - Only include questions whose answers materially impact ML architecture, pipeline design, task decomposition, ML testing strategy, UX behavior, or ML compliance validation.
   - Focus on ML-specific concerns: framework compatibility, resource requirements, pipeline patterns, model management.

4. Sequential questioning loop (interactive):
    - Present EXACTLY ONE question at a time.
    - For multiple‑choice questions render options as a Markdown table.
    - After the user answers, validate and record in working memory.
    - Stop when all critical ML ambiguities resolved or 5 questions reached.

5. Integration after EACH accepted answer (incremental update approach):
    - Ensure a `## Clarifications` section exists.
    - Under it, create a `### Session YYYY-MM-DD` subheading for today.
    - Append: `- Q: <question> → A: <final answer>`.
    - Apply clarification to appropriate sections:
       * ML workflow ambiguity → Update ML Pipeline Requirements
       * Component interaction → Update Pipeline Architecture
       * Framework compatibility → Update Technical Requirements
       * Resource constraints → Update Performance Requirements
    - Save the spec file AFTER each integration.

6. Report completion:
   - Number of questions asked & answered.
   - Path to updated spec.
   - Sections touched.
   - Coverage summary for ML pipeline categories.
   - Recommendation for next steps.

Focus on ML pipeline patterns, KFP-specific terminology, and Kubernetes-native ML workflows throughout the clarification process.