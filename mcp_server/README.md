1. Install dependencies:
```bash
pip install -r requirements.txt
```

2. Run the MCP server:
```bash
python mcp_pipeline_server.py
```

## Available Tools

### 1. `create_component`
Create a pipeline component based on requirements.

**Parameters:**
- `requirements` (string, required): Description of what the component should do
- `component_name` (string, optional): Specific component name to use
- `random` (boolean, optional): Whether to create a random component

**Example:**
```json
{
  "requirements": "Load and preprocess CSV data for machine learning",
  "component_name": "data_loader"
}
```

### 2. `list_components`
List all available predefined components.

**Example Response:**
```json
{
  "data_loader": {
    "description": "Load data from various sources",
    "inputs": 2,
    "outputs": 1
  },
  "model_trainer": {
    "description": "Train a machine learning model",
    "inputs": 3,
    "outputs": 1
  }
}
```

### 3. `create_pipeline_spec`
Create a complete pipeline specification.

**Parameters:**
- `name` (string, required): Pipeline name
- `description` (string, required): Pipeline description
- `components` (array, required): List of component names to include
- `requirements` (string, required): Overall pipeline requirements

**Example:**
```json
{
  "name": "ml-training-pipeline",
  "description": "Complete ML training pipeline",
  "components": ["data_loader", "data_preprocessor", "model_trainer"],
  "requirements": "Train a machine learning model on CSV data"
}
```

### 4. `generate_pipeline_yaml`
Generate Kubeflow Pipeline YAML from a pipeline specification.

**Parameters:**
- `pipeline_spec` (object, required): Pipeline specification from `create_pipeline_spec`

### 5. `create_complete_pipeline`
Create a complete pipeline from requirements (combines all steps).

**Parameters:**
- `name` (string, required): Pipeline name
- `description` (string, required): Pipeline description
- `requirements` (string, required): Detailed requirements
- `num_components` (integer, optional): Number of components (default: 3)

**Example:**
```json
{
  "name": "fraud-detection-pipeline",
  "description": "Pipeline to detect fraudulent transactions",
  "requirements": "Load transaction data, preprocess and clean it, train a fraud detection model, and evaluate performance",
  "num_components": 4
}
```

## Predefined Components

The server includes several predefined components:

### data_loader
- **Purpose**: Load data from various sources
- **Image**: `python:3.9-slim`
- **Inputs**: input_path, output_path
- **Outputs**: output_data

### data_preprocessor
- **Purpose**: Preprocess and clean data
- **Image**: `python:3.9-slim`
- **Inputs**: input_data, output_path
- **Outputs**: preprocessed_data

### model_trainer
- **Purpose**: Train a machine learning model
- **Image**: `tensorflow/tensorflow:2.13.0`
- **Inputs**: training_data, model_type, model_output_path
- **Outputs**: trained_model

### model_evaluator
- **Purpose**: Evaluate a trained model
- **Image**: `python:3.9-slim`
- **Inputs**: model_path, test_data, metrics_output_path
- **Outputs**: evaluation_metrics

## Usage Examples

### Example 1: Create a Simple Data Processing Component

```json
{
  "tool": "create_component",
  "arguments": {
    "requirements": "Load CSV data and perform basic cleaning",
    "component_name": "data_loader"
  }
}
```

### Example 2: Generate a Complete ML Pipeline

```json
{
  "tool": "create_complete_pipeline",
  "arguments": {
    "name": "customer-churn-prediction",
    "description": "Predict customer churn using historical data",
    "requirements": "Load customer data, preprocess features, train a classification model, and evaluate accuracy"
  }
}
```

### Example 3: Create Custom Pipeline with Specific Components

```json
{
  "tool": "create_pipeline_spec",
  "arguments": {
    "name": "image-classification-pipeline",
    "description": "Train an image classification model",
    "components": ["data_loader", "image_preprocessor", "cnn_trainer", "model_evaluator"],
    "requirements": "Process images and train a CNN for classification"
  }
}
```

## Smart Component Selection

The server intelligently selects components based on keywords in requirements:

- **Data keywords** (`data`, `load`, `extract`) → data processing components
- **ML keywords** (`train`, `model`, `ml`, `ai`) → model training components
- **Preprocessing keywords** (`preprocess`, `clean`, `transform`) → data preprocessing
- **Evaluation keywords** (`eval`, `test`, `validate`) → model evaluation
- **Visualization keywords** (`visual`, `plot`, `chart`) → visualization components

## Generated Pipeline Structure

The server generates pipelines in Kubeflow's Argo Workflow format:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: pipeline-name-
  annotations:
    pipelines.kubeflow.org/kfp_sdk_version: "2.1.0"
spec:
  entrypoint: pipeline
  templates:
    # Component templates
    - name: component-name
      container:
        image: python:3.9-slim
        command: [python]
        args: ["-c", "processing script"]
    # Main pipeline DAG
    - name: pipeline
      dag:
        tasks:
          - name: task-1
            template: component-1
          - name: task-2
            template: component-2
            dependencies: [task-1]
```

## Resource Management

Components include resource requests and limits:

```json
{
  "resource_requests": {
    "cpu": "100m",
    "memory": "256Mi"
  },
  "resource_limits": {
    "cpu": "1000m",
    "memory": "1Gi"
  }
}
```

## Error Handling

The server includes comprehensive error handling:

- Invalid tool names
- Missing required parameters
- Component generation failures
- YAML serialization errors

All errors are returned with descriptive messages and error flags.

## Extending the Server

To add new predefined components:

1. Add to `_load_predefined_components()` method
2. Define component with proper inputs/outputs
3. Include appropriate Docker image and commands

To modify component selection logic:

1. Update `create_random_component()` method
2. Add new keyword patterns
3. Include new Docker images in `common_images`

## Integration

This MCP server can be integrated with:

- Claude Desktop
- Other MCP-compatible clients
- Custom applications using MCP protocol
- CI/CD pipelines for automated pipeline generation

## License

MIT License - See LICENSE file for details.
