# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Kubeflow Pipelines (KFP) is a machine learning workflow orchestration platform built on Kubernetes and Argo Workflows. This is a multi-language monorepo with Go backend services, Python SDK, and React/TypeScript frontend.

**Current Status**: v2 engine (mature), with legacy v1 support. Main branch: `master`.

## Architecture

The system follows a compilation-then-execution model:

```
Python DSL → Pipeline IR (YAML) → Argo Workflow → Kubernetes Pods
```

### Key Components

- **SDK (Python)**: `sdk/python/kfp/` - DSL, compiler, client, local execution
- **API Server (Go)**: `backend/src/apiserver/` - REST/gRPC API, workflow submission
- **Compiler (Go)**: `backend/src/v2/compiler/` - Converts pipeline IR to Argo Workflows
- **Driver/Launcher (Go)**: `backend/src/v2/cmd/` - Input resolution, artifact I/O
- **Frontend (React/TS)**: `frontend/` - Web UI for pipeline management

### Critical Path Files

- Pipeline compilation: `sdk/python/kfp/compiler/pipeline_spec_builder.py`
- Executor runtime: `sdk/python/kfp/dsl/executor_main.py`
- API definitions: `api/v2alpha1/pipeline_spec.proto`
- Architecture diagram: `images/kfp-cluster-wide-architecture.png`

## Development Commands

### Environment Setup

```bash
# Python SDK development
python3 -m venv .venv && source .venv/bin/activate
python -m pip install -U pip setuptools wheel

make -C api python-dev
make -C kubernetes_platform python-dev
pip install -e api/v2alpha1/python --config-settings editable_mode=strict
pip install -e sdk/python --config-settings editable_mode=strict
pip install -e kubernetes_platform/python --config-settings editable_mode=strict

# Install ginkgo for Go test suites
make ginkgo
export PATH="$PWD/bin:$PATH"
```

### Testing

```bash
# Python SDK tests
pytest -v sdk/python/kfp

# Go backend unit tests (excludes integration tests)
go test -v $(go list ./backend/... | grep -v backend/test/v2/api | grep -v backend/test/integration | grep -v backend/test/end2end | grep -v backend/test/compiler)

# Compiler tests (Go/Ginkgo)
ginkgo -v ./backend/test/compiler
ginkgo -v ./backend/test/compiler -- -updateCompiledFiles=true  # Update goldens

# API integration tests (Go/Ginkgo, requires cluster)
ginkgo -v --label-filter="Smoke" ./backend/test/v2/api

# E2E tests (requires cluster)
ginkgo -v ./backend/test/end2end -- -namespace=kubeflow
```

### Build and Deploy

```bash
# Local cluster (standalone mode)
make -C backend kind-cluster-agnostic

# Development cluster (with debugging support)
make -C backend dev-kind-cluster

# Backend Docker images
make -C backend all

# Frontend development
cd frontend
npm ci
npm run mock:api           # Mock backend
npm start                  # Dev server
npm run start:proxy-and-server  # With cluster proxy
```

### Code Generation

```bash
# Regenerate protobufs (after .proto changes)
make -C api python && make -C api golang

# Frontend API clients
cd frontend && npm run apis

# Note: SELinux enforcing may break protoc; use `sudo setenforce 0` temporarily
```

## Just Recipes (Optional)

Convenience wrappers around Make targets:

```bash
just                      # List all recipes
just backend-images       # Build Docker images
just backend-test         # Go unit tests
just api-protos          # Generate protobuf code
just kind-standalone     # Deploy local cluster
```

## File Structure

```
├── sdk/python/kfp/           # Primary Python SDK
│   ├── compiler/             # DSL → Pipeline IR compilation
│   ├── dsl/                  # Domain-specific language core
│   ├── client/               # API client for remote execution
│   └── local/                # Local execution runners
├── backend/src/              # Go backend services
│   ├── apiserver/            # Main API server
│   ├── v2/                   # V2 engine (current)
│   └── cache/                # Task caching service
├── api/v2alpha1/             # Protobuf API contracts
├── kubernetes_platform/     # K8s-specific extensions
├── frontend/                 # React/TypeScript UI
├── manifests/kustomize/      # Kubernetes deployment configs
└── test_data/                # Test fixtures and golden files
```

## Development Patterns

### Pipeline Execution Flow

1. **SDK Compilation**: Python DSL → Pipeline IR YAML
2. **API Submission**: Client uploads pipeline spec to API server
3. **Backend Compilation**: Pipeline IR → Argo Workflow
4. **Runtime Execution**: Driver resolves inputs → Launcher handles I/O → Python executor runs task

### Local Development Modes

- **Subprocess Runner**: Direct Python execution (lightweight components only)
- **Docker Runner**: Container execution (all component types)
- **Remote Execution**: Submit to Kubernetes cluster

### Generated Files (Never Edit Directly)

- `api/v2alpha1/python/kfp/pipeline_spec/pipeline_spec_pb2.py` ← `pipeline_spec.proto`
- `kubernetes_platform/python/kfp/kubernetes/kubernetes_executor_config_pb2.py` ← K8s proto
- `frontend/src/apis/*` ← Backend Swagger specs
- `frontend/src/third_party/mlmd/generated/*` ← ML Metadata protos

### Environment Variables

- `_KFP_RUNTIME=true`: Disables SDK imports during task execution
- `LOCAL_API_SERVER=true`: Local API server testing mode

## Key Dependencies

| Component | Technology | Versions |
|-----------|------------|----------|
| Backend | Go | 1.24.10 |
| SDK | Python | 3.9-3.13 |
| Frontend | Node.js | 22.14.0 |
| Orchestration | Argo Workflows | v3.5-v3.7 |
| Database | MySQL | v8 |

## Troubleshooting

- **Protobuf generation fails**: Check SELinux (`setenforce 0` temporarily), ensure `protoc` available
- **`_KFP_RUNTIME=true` import errors**: Avoid importing SDK-only modules in task code
- **Frontend API generation**: Requires `swagger-codegen-cli.jar` in PATH
- **Node version issues**: Use `fnm use` or `nvm use $(cat frontend/.nvmrc)`
- **Port conflicts**: Frontend uses 3000 (React), 3001 (Node), 3002 (proxy), 6006 (Storybook)

## Essential Documentation

- `/AGENTS.md` - Comprehensive developer guide (your primary reference)
- `/developer_guide.md` - Build and deployment instructions
- `/docs/sdk/Architecture.md` - System architecture overview
- `/frontend/README.md` - Frontend development guide
- `/backend/README.md` - Backend development guide

## Code Quality Standards

Follow the software engineering principles outlined in `.github/copilot-instructions.md`:
- **SOLID principles** for design
- **DRY, KISS, YAGNI** for implementation
- **High cohesion, low coupling** for architecture
- Strong type hints for Python, clear docstrings, comprehensive unit tests
- **Boy Scout Rule**: Leave code cleaner than you found it
- Be concise and precise with code comments
- Do not try to add too much content to documentation - be concise and precise

## Self-Review Protocol:                                                                                                                                                                                   │
│   Before presenting your review:                                                                                                                                                                          │
│   1. Re-read your analysis as if you're reviewing the reviewer                                                                                                                                            │
│   2. Double-check that you've identified all potential issues                                                                                                                                             │
│   3. Verify your suggestions are actionable and follow senior developer standards                                                                                                                         │
│   4. Ensure you haven't missed any security vulnerabilities or edge cases                                                                                                                                 │
│   5. If you find and fix issues in your own analysis, note: "Self-review: Fixed [issue]"  