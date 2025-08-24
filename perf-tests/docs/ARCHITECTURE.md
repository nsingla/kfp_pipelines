# Performance Test Framework Architecture

This document provides a detailed technical overview of the KFP Performance Test Framework architecture, design patterns, and implementation details.

## Design Patterns

### 1. Template Method Pattern

The framework uses the Template Method pattern through the `BaseRunner` abstract class:

```python
class BaseRunner:
    def __init__(self, test_scenario: TestScenario):
        # Common initialization logic
        self.setup_kfp_client()
        self.calculate_timing()
    
    def start(self):
        # Common start logic - wait for scheduled time
        self.wait_for_start_time()
    
    @abstractmethod
    def run(self):
        # Template method - subclasses implement specific logic
        pass
    
    @abstractmethod
    def stop(self):
        # Template method - subclasses implement specific logic
        pass
```

**Benefits:**
- Enforces consistent initialization and timing across all runners
- Allows for specialized behavior in subclasses
- Reduces code duplication

### 2. Strategy Pattern

Different test modes are implemented as separate runner strategies:

```python
# Strategy selection in test_performance.py
if scenario.mode == TestMode.PIPELINE_RUN or scenario.mode == TestMode.EXPERIMENT:
    runner = PipelineRunner(scenario)
elif scenario.mode == TestMode.RANDOM_GETS:
    runner = RandomGetsRunner(scenario)  # Future implementation
```

**Benefits:**
- Easy to add new test modes
- Runtime strategy selection
- Clean separation of concerns

### 3. Factory Pattern

The test execution uses a factory-like approach to create runners:

```python
def create_runner(scenario: TestScenario) -> BaseRunner:
    if scenario.mode == TestMode.PIPELINE_RUN:
        return PipelineRunner(scenario)
    elif scenario.mode == TestMode.EXPERIMENT:
        return PipelineRunner(scenario)  # Same runner, different behavior
    # Add more runners as needed
```

## Class Relationships

### Inheritance Hierarchy

```
BaseRunner (Abstract)
├── PipelineRunner
├── ExperimentRunner (Future)
└── RandomGetsRunner (Future)
```

### Composition Relationships

```
TestPerformance
├── ThreadPoolExecutor
│   ├── PipelineRunner
│   │   ├── TestScenario
│   │   ├── KFP Client
│   │   └── Logger
│   └── OtherRunners...
└── JsonDeserializationUtils
    └── TestScenario[]
```

### Dependency Injection

The framework uses dependency injection for:
- **Configuration**: Environment-based configuration injection
- **Logging**: Centralized logger injection
- **KFP Client**: Client injection for different deployment modes

## Concurrency Model

### Thread Pool Execution

```python
with ThreadPoolExecutor(max_workers=len(scenarios)) as executor:
    futures = []
    for scenario in scenarios:
        runner = create_runner(scenario)
        futures.append(executor.submit(runner.run))
```

**Characteristics:**
- **Fixed Thread Pool**: One thread per test scenario
- **Non-blocking**: Uses `asyncio` within threads for I/O operations
- **Result Collection**: `as_completed()` for handling results as they finish

### Async/Await Pattern

The `PipelineRunner` uses async/await for non-blocking operations:

```python
async def wait_for_run_to_finish(self, run_id: str):
    while pipeline_state in [RUNNING, CANCELING, PENDING]:
        await asyncio.sleep(self.polling_wait_time_for_run_to_finish)
        pipeline_state = self.client.get_run(run_id).state
```

**Benefits:**
- Non-blocking I/O operations
- Efficient resource utilization
- Better responsiveness

## Data Flow

### 1. Configuration Loading

```
Environment Variables → TestConfig → Runner Initialization
```

### 2. Scenario Deserialization

```
JSON File → JsonDeserializationUtils → TestScenario[] → Runner Creation
```

### 3. Test Execution

```
Runner.run() → Pipeline Upload → Pipeline Execution → Metrics Collection → Results
```

### 4. Metrics Collection

```
Pipeline Runs → Execution Times → Metrics Dictionary → Test Results
```

## Error Handling Strategy

### 1. Exception Propagation

```python
try:
    # Test execution
    for future in as_completed(futures):
        result = future.result()
except Exception as e:
    test_failed = True
    exception_to_throw = e

if test_failed:
    raise exception_to_throw
```

### 2. Graceful Degradation

- Individual runner failures don't stop other runners
- Partial results are still collected
- Comprehensive error logging

### 3. Resource Cleanup

- Automatic thread pool cleanup with context manager
- Pipeline resource cleanup in runner implementations
- Log file rotation and cleanup

## Configuration Management

### Environment-Based Configuration

```python
class TestConfig:
    KFP_API_HOST: str = os.getenv("API_HOST", "localhost")
    KFP_API_PORT: str = os.getenv("API_PORT", "8888")
    IS_KUBEFLOW_MODE: bool = bool(os.getenv("isKubeflowMode", "False"))
```

**Benefits:**
- Environment-specific configuration
- No hardcoded values
- Easy deployment across environments

### Configuration Validation

The framework uses Pydantic for configuration validation:

```python
class TestScenario(BaseModel):
    model_config = ConfigDict(extra='forbid', arbitrary_types_allowed=True)
    mode: EnumSerDe(TestMode).enum_by_name()
    start_time: int = Field(alias="startTime")
    run_time: int = Field(alias="runTime")
```

**Features:**
- Automatic type conversion
- Field validation
- Error messages for invalid configurations

## Logging Architecture

### Multi-Handler Logging

```python
class Logger:
    def _configure_logger(self, logger: logging.Logger):
        # Console handler with colors
        self._configure_console_handler(logger)
        # File handler with rotation
        self._configure_file_handler(logger)
        # JSON handler for structured logging
        self._configure_json_handler(logger)
```

### Log Levels and Colors

```python
LOG_COLORS = {
    "INFO": "green",
    "REQUESTS": "bold_white",
    "DEBUG": "light_white",
    "WARNING": "yellow",
    "ERROR": "red",
    "CRITICAL": "red"
}
```

### Thread-Safe Logging

- Singleton logger instance
- Thread-safe handlers
- Rotating file handlers with size limits

## Performance Considerations

### 1. Memory Management

- **Lazy Loading**: Pipeline files loaded only when needed
- **Resource Cleanup**: Automatic cleanup of KFP resources
- **Log Rotation**: Prevents log files from growing indefinitely

### 2. Network Efficiency

- **Connection Reuse**: KFP client maintains persistent connections
- **Async Operations**: Non-blocking I/O for better throughput
- **Polling Optimization**: Configurable polling intervals

### 3. Scalability

- **Horizontal Scaling**: Multiple scenarios run in parallel
- **Vertical Scaling**: Configurable thread pool sizes
- **Resource Limits**: Configurable timeouts and retry limits

## Extension Points

### 1. New Test Modes

```python
class CustomRunner(BaseRunner):
    def run(self):
        # Custom test logic
        pass
    
    def stop(self):
        # Custom cleanup logic
        pass
```

### 2. New Metrics

```python
class MetricsCollector:
    def collect_custom_metrics(self):
        # Custom metric collection
        return {"custom_metric": value}
```

### 3. New Configuration Options

```python
class ExtendedTestConfig(TestConfig):
    CUSTOM_SETTING: str = os.getenv("CUSTOM_SETTING", "default")
```

## Testing Strategy

### 1. Unit Testing

- Individual component testing
- Mock KFP client for isolated testing
- Configuration validation testing

### 2. Integration Testing

- End-to-end scenario testing
- Real KFP deployment testing
- Performance regression testing

### 3. Load Testing

- Concurrent scenario execution
- Resource utilization testing
- Scalability validation

## Security Considerations

### 1. SSL/TLS Configuration

```python
verify_ssl = False if TestConfig.LOCAL else True
self.client = kfp.Client(host=TestConfig.KFP_url, verify_ssl=verify_ssl)
```

### 2. Namespace Isolation

- Configurable namespace for multi-tenant deployments
- Resource isolation between test runs

### 3. Credential Management

- Environment-based credential injection
- No hardcoded secrets in code

## Monitoring and Observability

### 1. Metrics Collection

- Execution time tracking
- Success/failure rates
- Resource utilization metrics

### 2. Structured Logging

- JSON format logs for easy parsing
- Correlation IDs for request tracing
- Log aggregation support

### 3. Health Checks

- KFP API connectivity checks
- Resource availability monitoring
- Test execution status tracking 