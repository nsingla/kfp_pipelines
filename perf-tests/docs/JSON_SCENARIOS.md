# JSON Test Scenario Configuration Guide

This document provides comprehensive guidance on creating and using JSON test scenario files for the KFP Performance Test Framework.

## Overview

JSON test scenario files are the primary configuration mechanism for defining performance test behavior. They allow you to specify test parameters, timing, concurrency, and execution modes without modifying code.

## File Structure

### Basic Schema

```json
[
  {
    "mode": "PIPELINE_RUN",
    "startTime": 0,
    "runTime": 5,
    "numTimes": 2,
    "pipelineFileName": "add_numbers.yaml"
  }
]
```

### Advanced Schema with Multiple Scenarios

```json
[
  {
    "mode": "EXPERIMENT",
    "startTime": 0,
    "runTime": 10,
    "numTimes": 5,
    "pipelineFileName": "ml_training.yaml"
  },
  {
    "mode": "PIPELINE_RUN",
    "startTime": 2,
    "runTime": 8,
    "numTimes": 3,
    "pipelineFileName": "data_processing.yaml"
  }
]
```

## Field Reference

### Required Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `mode` | string | Test execution mode | `"PIPELINE_RUN"` |
| `startTime` | integer | Minutes to wait before starting | `0` |
| `runTime` | integer | Minutes to run the test | `5` |

### Optional Fields

| Field | Type | Description | Default | Example |
|-------|------|-------------|---------|---------|
| `numTimes` | integer | Number of pipeline runs | `1` | `5` |
| `pipelineFileName` | string | Pipeline YAML file name | `null` | `"my_pipeline.yaml"` |

## Test Modes

### 1. PIPELINE_RUN

Executes pipeline runs in the default experiment.

**Use Cases:**
- Basic pipeline performance testing
- Load testing with existing experiments
- Regression testing

**Example:**
```json
{
  "mode": "PIPELINE_RUN",
  "startTime": 0,
  "runTime": 5,
  "numTimes": 3,
  "pipelineFileName": "simple_pipeline.yaml"
}
```

**Behavior:**
- Uses the "default" experiment
- Uploads pipeline if `pipelineFileName` is specified
- Creates `numTimes` pipeline runs
- Runs for `runTime` minutes

### 2. EXPERIMENT

Creates a new experiment and runs pipelines within it.

**Use Cases:**
- Isolated testing environments
- Experiment-specific performance testing
- Multi-tenant testing scenarios

**Example:**
```json
{
  "mode": "EXPERIMENT",
  "startTime": 1,
  "runTime": 10,
  "numTimes": 5,
  "pipelineFileName": "complex_pipeline.yaml"
}
```

**Behavior:**
- Creates a new experiment with unique name
- Uploads pipeline if `pipelineFileName` is specified
- Creates `numTimes` pipeline runs in the new experiment
- Runs for `runTime` minutes

### 3. RANDOM_GETS (Future)

Performs random GET operations against the KFP API.

**Note:** This mode is planned for future implementation.

## Timing Configuration

### Start Time (`startTime`)

Defines when the test should begin relative to the test framework start.

**Examples:**
```json
// Start immediately
"startTime": 0

// Start after 2 minutes
"startTime": 2

// Start after 30 minutes
"startTime": 30
```

**Use Cases:**
- **Staggered Start**: Different scenarios start at different times
- **Warm-up Period**: Allow system to stabilize before testing
- **Sequential Testing**: Run tests in specific order

### Run Time (`runTime`)

Defines how long the test should execute.

**Examples:**
```json
// Run for 5 minutes
"runTime": 5

// Run for 1 hour
"runTime": 60

// Run for 24 hours
"runTime": 1440
```

**Considerations:**
- Longer run times provide more stable metrics
- Shorter run times are good for quick validation
- Balance between test duration and resource usage

## Concurrency Configuration

### Number of Times (`numTimes`)

Defines how many pipeline runs to execute during the test period.

**Examples:**
```json
// Single pipeline run
"numTimes": 1

// Multiple pipeline runs
"numTimes": 5

// High concurrency
"numTimes": 20
```

**Behavior:**
- The runner will maintain `numTimes` concurrent pipeline runs
- If a run completes, a new one is started
- This continues for the duration of `runTime`

## Pipeline Configuration

### Pipeline File Name (`pipelineFileName`)

Specifies which pipeline YAML file to use for testing.

**File Location:**
```
test_data/pipeline_files/{pipelineFileName}
```

**Examples:**
```json
// Use a simple pipeline
"pipelineFileName": "add_numbers.yaml"

// Use a complex ML pipeline
"pipelineFileName": "ml_training_pipeline.yaml"

// Use a data processing pipeline
"pipelineFileName": "etl_pipeline.yaml"
```

**Requirements:**
- File must exist in `test_data/pipeline_files/`
- Must be valid KFP pipeline YAML
- Should be appropriate for performance testing

## Scenario Examples

### 1. Smoke Test

Quick validation of basic functionality.

```json
[
  {
    "mode": "PIPELINE_RUN",
    "startTime": 0,
    "runTime": 2,
    "numTimes": 1,
    "pipelineFileName": "simple_pipeline.yaml"
  }
]
```

### 2. Load Test

High concurrency testing.

```json
[
  {
    "mode": "PIPELINE_RUN",
    "startTime": 0,
    "runTime": 30,
    "numTimes": 10,
    "pipelineFileName": "medium_pipeline.yaml"
  }
]
```

### 3. Stress Test

Maximum load testing.

```json
[
  {
    "mode": "PIPELINE_RUN",
    "startTime": 0,
    "runTime": 60,
    "numTimes": 50,
    "pipelineFileName": "lightweight_pipeline.yaml"
  }
]
```

### 4. Multi-Scenario Test

Testing different scenarios simultaneously.

```json
[
  {
    "mode": "EXPERIMENT",
    "startTime": 0,
    "runTime": 20,
    "numTimes": 5,
    "pipelineFileName": "ml_pipeline.yaml"
  },
  {
    "mode": "PIPELINE_RUN",
    "startTime": 5,
    "runTime": 15,
    "numTimes": 3,
    "pipelineFileName": "data_pipeline.yaml"
  }
]
```

### 5. Staggered Load Test

Gradual increase in load.

```json
[
  {
    "mode": "PIPELINE_RUN",
    "startTime": 0,
    "runTime": 30,
    "numTimes": 2,
    "pipelineFileName": "test_pipeline.yaml"
  },
  {
    "mode": "PIPELINE_RUN",
    "startTime": 10,
    "runTime": 20,
    "numTimes": 5,
    "pipelineFileName": "test_pipeline.yaml"
  },
  {
    "mode": "PIPELINE_RUN",
    "startTime": 20,
    "runTime": 10,
    "numTimes": 10,
    "pipelineFileName": "test_pipeline.yaml"
  }
]
```

## Best Practices

### 1. Pipeline Selection

**Choose Appropriate Pipelines:**
- Use lightweight pipelines for high-concurrency tests
- Use realistic pipelines for production-like testing
- Avoid pipelines with external dependencies for basic testing

**Pipeline Characteristics:**
```yaml
# Good for performance testing
- Short execution time (< 5 minutes)
- Minimal resource requirements
- No external API calls
- Deterministic behavior

# Avoid for performance testing
- Long-running pipelines (> 30 minutes)
- Heavy resource usage
- External dependencies
- Non-deterministic behavior
```

### 2. Timing Configuration

**Start Time Strategy:**
- Use `startTime: 0` for immediate execution
- Use staggered start times for load testing
- Allow warm-up time for production environments

**Run Time Strategy:**
- Minimum 5 minutes for meaningful results
- 15-30 minutes for load testing
- 60+ minutes for stress testing

### 3. Concurrency Configuration

**Number of Times Strategy:**
- Start with low concurrency (1-5)
- Gradually increase for load testing
- Monitor system resources during testing

**Concurrency Guidelines:**
```json
// Light load
"numTimes": 1-5

// Medium load
"numTimes": 5-20

// Heavy load
"numTimes": 20-50

// Stress testing
"numTimes": 50+
```

### 4. Scenario Organization

**File Naming:**
```
scenarios/
├── smoke.json          # Quick validation
├── load_test.json      # Medium load testing
├── stress_test.json    # High load testing
├── regression.json     # Regression testing
└── custom_scenario.json # Custom scenarios
```

**Scenario Structure:**
- Keep scenarios focused on specific testing goals
- Use descriptive names for scenarios
- Document scenario purpose in comments

## Validation and Troubleshooting

### JSON Validation

**Common Issues:**
```json
// ❌ Invalid - missing required field
{
  "mode": "PIPELINE_RUN",
  "runTime": 5
}

// ✅ Valid - all required fields present
{
  "mode": "PIPELINE_RUN",
  "startTime": 0,
  "runTime": 5,
  "numTimes": 1
}
```

**Validation Checklist:**
- [ ] All required fields present
- [ ] Field types are correct
- [ ] Mode value is valid
- [ ] Timing values are positive integers
- [ ] Pipeline file exists (if specified)

### Common Errors

**1. Invalid Mode**
```json
// ❌ Error: Invalid mode
{
  "mode": "INVALID_MODE",
  "startTime": 0,
  "runTime": 5
}
```

**2. Missing Pipeline File**
```json
// ❌ Error: Pipeline file not found
{
  "mode": "PIPELINE_RUN",
  "startTime": 0,
  "runTime": 5,
  "pipelineFileName": "nonexistent.yaml"
}
```

**3. Invalid Timing**
```json
// ❌ Error: Negative timing
{
  "mode": "PIPELINE_RUN",
  "startTime": -1,
  "runTime": 5
}
```

### Debugging Tips

**1. Check File Paths:**
```bash
# Verify pipeline file exists
ls test_data/pipeline_files/your_pipeline.yaml

# Verify scenario file exists
ls test_data/scenarios/your_scenario.json
```

**2. Validate JSON Syntax:**
```bash
# Check JSON syntax
python -m json.tool test_data/scenarios/your_scenario.json
```

**3. Test with Simple Scenario:**
```json
# Start with minimal configuration
[
  {
    "mode": "PIPELINE_RUN",
    "startTime": 0,
    "runTime": 1,
    "numTimes": 1
  }
]
```

## Environment-Specific Configuration

### Development Environment

```json
[
  {
    "mode": "PIPELINE_RUN",
    "startTime": 0,
    "runTime": 2,
    "numTimes": 1,
    "pipelineFileName": "simple_pipeline.yaml"
  }
]
```

### Staging Environment

```json
[
  {
    "mode": "PIPELINE_RUN",
    "startTime": 0,
    "runTime": 10,
    "numTimes": 5,
    "pipelineFileName": "test_pipeline.yaml"
  }
]
```

### Production Environment

```json
[
  {
    "mode": "EXPERIMENT",
    "startTime": 5,
    "runTime": 30,
    "numTimes": 10,
    "pipelineFileName": "production_pipeline.yaml"
  }
]
```

## Advanced Configuration

### Conditional Scenarios

Use environment variables to conditionally include scenarios:

```bash
# Set environment variable
export TEST_MODE=load

# Use in scenario selection
if [ "$TEST_MODE" = "load" ]; then
  SCENARIO_FILE=load_test.json
else
  SCENARIO_FILE=smoke.json
fi
```

### Dynamic Configuration

Generate scenarios programmatically:

```python
import json

def create_load_test_scenario(concurrency, duration):
    return {
        "mode": "PIPELINE_RUN",
        "startTime": 0,
        "runTime": duration,
        "numTimes": concurrency,
        "pipelineFileName": "test_pipeline.yaml"
    }

scenarios = [
    create_load_test_scenario(5, 10),
    create_load_test_scenario(10, 15)
]

with open("dynamic_scenario.json", "w") as f:
    json.dump(scenarios, f, indent=2)
``` 