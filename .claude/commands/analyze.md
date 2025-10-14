---
description: Perform a non-destructive cross-artifact consistency and quality analysis across spec.md, plan.md, and tasks.md after task generation for KFP pipeline features.
---

The user input to you can be provided directly by the agent or as a command argument - you **MUST** consider it before proceeding with the prompt (if not empty).

User input:

$ARGUMENTS

Goal: Identify inconsistencies, duplications, ambiguities, and underspecified items across the three core artifacts (`spec.md`, `plan.md`, `tasks.md`) before ML pipeline implementation. This command MUST run only after `/tasks` has successfully produced a complete `tasks.md`.

STRICTLY READ-ONLY: Do **not** modify any files. Output a structured analysis report. Offer an optional remediation plan (user must explicitly approve before any follow-up editing commands would be invoked manually).

Constitution Authority: The project constitution (`rfe/memory/constitution.md`) is **non-negotiable** within this analysis scope. Constitution conflicts are automatically CRITICAL and require adjustment of the spec, plan, or tasks—not dilution, reinterpretation, or silent ignoring of the principle.

Execution steps:

1. Run `rfe/scripts/check-prerequisites.sh --json --require-tasks --include-tasks` once from repo root and parse JSON for FEATURE_DIR and AVAILABLE_DOCS. Derive absolute paths:
   - SPEC = FEATURE_DIR/spec.md
   - PLAN = FEATURE_DIR/plan.md
   - TASKS = FEATURE_DIR/tasks.md
   Abort with an error message if any required file is missing.

2. Load artifacts:
   - Parse spec.md sections: Overview/Context, ML Pipeline Requirements, Non-Functional Requirements, User Stories, Edge Cases.
   - Parse plan.md: ML Architecture/stack choices, Pipeline Component Design, Implementation Phases, Technical constraints.
   - Parse tasks.md: Task IDs, descriptions, phase grouping, parallel markers [P], referenced ML components and file paths.
   - Load constitution `rfe/memory/constitution.md` for principle validation.

3. Build internal semantic models for ML pipelines:
   - ML requirements inventory: Each functional + non-functional requirement with a stable key
   - ML user story/workflow inventory
   - Pipeline task coverage mapping: Map each task to one or more ML requirements or workflows
   - Constitution rule set: Extract principle names and ML-specific MUST/SHOULD normative statements

4. Detection passes for ML pipeline features:
   A. Duplication detection:
      - Identify near-duplicate ML requirements or pipeline components
   B. Ambiguity detection:
      - Flag vague ML adjectives (scalable, robust, efficient) lacking measurable criteria
      - Flag unresolved ML placeholders (framework TBD, resource limits TBD, etc.)
   C. Underspecification for ML pipelines:
      - ML requirements with missing acceptance criteria
      - Pipeline components without clear input/output specifications
      - Tasks referencing ML frameworks or components not defined in spec/plan
   D. Constitution alignment:
      - Any ML requirement or plan element conflicting with MUST principles
      - Missing mandated ML testing or validation sections
   E. Coverage gaps for ML workflows:
      - ML requirements with zero associated implementation tasks
      - Tasks with no mapped ML requirement/story
      - Non-functional ML requirements not reflected in tasks (performance, security, compatibility)
   F. Inconsistency in ML context:
      - ML terminology drift (same concept named differently across files)
      - Pipeline components referenced in plan but absent in spec
      - ML framework versions or compatibility contradictions
      - Task ordering issues for ML pipeline dependencies

5. Severity assignment heuristic for ML features:
   - CRITICAL: Violates constitution MUST, missing core ML pipeline functionality, or requirement with zero coverage
   - HIGH: Duplicate or conflicting ML requirement, ambiguous performance/security attribute, untestable ML acceptance criterion
   - MEDIUM: ML terminology drift, missing ML framework compatibility task, underspecified ML edge case
   - LOW: Style/wording improvements for ML content, minor redundancy not affecting ML execution

6. Produce a Markdown report (no file writes) with sections:

   ### ML Pipeline Specification Analysis Report
   | ID | Category | Severity | Location(s) | Summary | Recommendation |
   |----|----------|----------|-------------|---------|----------------|
   | A1 | Duplication | HIGH | spec.md:L120-134 | Two similar ML pipeline requirements ... | Merge phrasing; keep clearer version |

   Additional subsections:
   - ML Pipeline Coverage Summary Table:
     | ML Requirement Key | Has Task? | Task IDs | Notes |
   - Constitution Alignment Issues (if any)
   - Unmapped ML Tasks (if any)
   - ML Pipeline Metrics:
     * Total ML Requirements
     * Total Implementation Tasks
     * Coverage % (ML requirements with >=1 task)
     * ML Ambiguity Count
     * ML Duplication Count
     * Critical Issues Count

7. Next Actions block for ML pipeline implementation:
   - If CRITICAL issues exist: Recommend resolving before ML implementation
   - If only LOW/MEDIUM: User may proceed with ML development
   - Provide ML-specific command suggestions

8. Ask the user: "Would you like me to suggest concrete remediation edits for the top N ML pipeline issues?"

Behavior rules:
- NEVER modify files.
- Focus on ML pipeline patterns, KFP component design, and Kubernetes-native workflows.
- Keep findings deterministic for consistent ML analysis.
- If zero issues found, emit a success report for ML pipeline readiness.

Context: $ARGUMENTS