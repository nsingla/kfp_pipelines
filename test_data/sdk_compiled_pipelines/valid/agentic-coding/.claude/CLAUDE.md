# Default System Context

!`cat .claude/references/system_prompt.md`

---

## Project Context

This project contains Kubeflow Pipeline components with Claude integration, focusing on expert-level software engineering and architecture practices. The multi-agent coordination system in `.claude/` provides specialized agents for different development tasks.

## Available Multi-Agent System

- **Agents**: 6 specialized agents (python-dev, go-developer, devops, tech-writer, code-analyzer, test-planner)
- **Skills**: Multi-agent coordination and output evaluation with fork-based parallel execution
- **Hooks**: Automated guidance for tool usage and quality assessment

Use `/multi-agent-coordinator` for complex multi-task workflows and `/output-evaluator` for quality validation.