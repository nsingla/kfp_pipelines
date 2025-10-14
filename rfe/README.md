# RFE (Request for Enhancement) Workflow System

This directory contains the RFE workflow system integrated into the Kubeflow Pipelines project, based on the vTeam architecture.

## Overview

The RFE workflow system provides an AI-powered, multi-agent approach to requirement analysis, design planning, and implementation task generation for Kubeflow Pipelines features.

## Architecture

### Core Components

1. **RFE Workflow Engine** (`workflow/`)
   - RFE lifecycle management
   - Agent council orchestration
   - Status tracking and transitions

2. **Agent System** (`agents/`)
   - 7-agent council implementation
   - Agent personas and capabilities
   - Multi-agent collaboration framework

3. **Document Generators** (`generators/`)
   - Spec.md generation (feature specifications)
   - Plan.md generation (implementation plans)
   - Tasks.md generation (actionable task lists)

4. **Integrations** (`integrations/`)
   - Git repository operations
   - Jira issue management
   - KFP-specific pipeline generation

### Agent Council

The RFE workflow uses a 7-agent council system:

1. **Parker (Product Manager)** - Business value and roadmap alignment
2. **Archie (Architect)** - Technical feasibility and system design
3. **Stella (Staff Engineer)** - Implementation complexity and approach
4. **Uma (UX Team Lead)** - User experience and design coordination
5. **Lee (Team Lead)** - Development coordination and resource planning
6. **Neil (Test Engineer)** - Testing strategy and quality assurance
7. **Jack (Delivery Owner)** - Cross-team coordination and delivery

## Workflow Process

1. **RFE Creation** - Initial feature request submission
2. **Agent Analysis** - Multi-agent evaluation and feedback
3. **Specification Generation** - Automated spec.md creation
4. **Implementation Planning** - Technical plan.md generation
5. **Task Breakdown** - Actionable tasks.md creation
6. **Approval Process** - Final review and acceptance
7. **Implementation Tracking** - Progress monitoring

## Usage

### Creating an RFE

```python
from rfe.workflow import RFEWorkflow

# Create new RFE
rfe = RFEWorkflow.create(
    title="Add support for XGBoost components",
    description="Enable XGBoost pipeline components for ML workflows",
    umbrella_repo="https://github.com/kubeflow/pipelines.git",
    supporting_repos=[
        "https://github.com/kubeflow/examples.git"
    ]
)

# Start agent analysis
rfe.start_analysis()

# Monitor progress
status = rfe.get_status()
```

### Integration with KFP

The RFE system integrates seamlessly with KFP development:

- **Pipeline Components** - Generate KFP component definitions
- **Workflow Templates** - Create Argo workflow templates
- **Test Automation** - Generate comprehensive test suites
- **Documentation** - Automatic API and user documentation

## Configuration

See `config/` directory for:
- Agent persona configurations
- Integration settings (Git, Jira, etc.)
- Workflow templates and patterns

## Development

### Prerequisites

- Python 3.8+
- Kubernetes cluster access
- Git and GitHub/GitLab integration
- Optional: Jira integration for issue tracking

### Setup

```bash
# Install dependencies
pip install -r requirements.txt

# Configure integrations
cp config/config.example.yaml config/config.yaml
# Edit config.yaml with your settings

# Run tests
pytest tests/
```

## Examples

See `examples/` directory for:
- Sample RFE workflows
- Agent interaction patterns
- Integration demonstrations
- KFP-specific use cases

## Contributing

This RFE system follows the standard KFP contribution guidelines. See the main project CONTRIBUTING.md for details.

## License

Same as Kubeflow Pipelines project license.