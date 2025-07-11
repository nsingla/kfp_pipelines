#!/usr/bin/env python3
"""MCP Server for Kubeflow Pipeline Generation.

This server provides tools to:
1. Create pipeline components (specific or random)
2. Generate pipeline specifications
3. Create complete pipelines

Usage:
    python mcp_pipeline_server.py
"""

from dataclasses import asdict
from dataclasses import dataclass
from datetime import datetime
import json
import logging
import random
from typing import Any, Dict, List, Optional
import uuid

from mcp.server import NotificationOptions
from mcp.server import Server
# MCP imports
from mcp.server.models import CallToolResult
from mcp.server.models import InitializationOptions
from mcp.server.models import Tool
from mcp.types import TextContent
import yaml

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger('pipeline-mcp-server')


# Data structures for pipeline components
@dataclass
class ComponentInput:
    name: str
    type: str
    description: str
    default: Optional[str] = None


@dataclass
class ComponentOutput:
    name: str
    type: str
    description: str


@dataclass
class PipelineComponent:
    name: str
    description: str
    image: str
    command: List[str]
    args: List[str]
    inputs: List[ComponentInput]
    outputs: List[ComponentOutput]
    resource_requests: Optional[Dict[str, str]] = None
    resource_limits: Optional[Dict[str, str]] = None


@dataclass
class PipelineTask:
    name: str
    component_ref: str
    inputs: Dict[str, Any]
    depends_on: List[str] = None


@dataclass
class PipelineSpec:
    name: str
    description: str
    components: List[PipelineComponent]
    tasks: List[PipelineTask]
    pipeline_spec_version: str = '2.1.0'


class PipelineGenerator:
    """Generator class for creating pipeline components and specs."""

    def __init__(self):
        self.predefined_components = self._load_predefined_components()
        self.common_images = [
            'python:3.9-slim',
            'gcr.io/deeplearning-platform-release/base-cpu',
            'tensorflow/tensorflow:2.13.0',
            'pytorch/pytorch:2.0.1-py3.9-cuda11.7-cudnn8-runtime',
            'jupyter/scipy-notebook:latest',
            'pandas/pandas:latest',
            'apache/spark:3.4.0',
        ]

    def _load_predefined_components(self) -> Dict[str, PipelineComponent]:
        """Load predefined common components."""
        return {
            'data_loader':
                PipelineComponent(
                    name='data-loader',
                    description='Load data from various sources',
                    image='python:3.9-slim',
                    command=['python'],
                    args=[
                        '-c',
                        'import pandas as pd; import sys; data = pd.read_csv(sys.argv[1]); data.to_csv(sys.argv[2], index=False)'
                    ],
                    inputs=[
                        ComponentInput('input_path', 'String',
                                       'Path to input data'),
                        ComponentInput('output_path', 'String',
                                       'Path to save processed data')
                    ],
                    outputs=[
                        ComponentOutput('output_data', 'Dataset',
                                        'Processed dataset')
                    ],
                    resource_requests={
                        'cpu': '100m',
                        'memory': '256Mi'
                    }),
            'data_preprocessor':
                PipelineComponent(
                    name='data-preprocessor',
                    description='Preprocess and clean data',
                    image='python:3.9-slim',
                    command=['python'],
                    args=[
                        '-c', """
import pandas as pd
import sys
data = pd.read_csv(sys.argv[1])
# Basic preprocessing
data = data.dropna()
data = data.drop_duplicates()
data.to_csv(sys.argv[2], index=False)
print(f'Processed {len(data)} rows')
"""
                    ],
                    inputs=[
                        ComponentInput('input_data', 'Dataset',
                                       'Raw dataset to preprocess'),
                        ComponentInput('output_path', 'String',
                                       'Path to save preprocessed data')
                    ],
                    outputs=[
                        ComponentOutput('preprocessed_data', 'Dataset',
                                        'Cleaned and preprocessed dataset')
                    ],
                    resource_requests={
                        'cpu': '200m',
                        'memory': '512Mi'
                    }),
            'model_trainer':
                PipelineComponent(
                    name='model-trainer',
                    description='Train a machine learning model',
                    image='tensorflow/tensorflow:2.13.0',
                    command=['python'],
                    args=[
                        '-c', """
import tensorflow as tf
import pandas as pd
import sys
import joblib

# Load data
data = pd.read_csv(sys.argv[1])
model_type = sys.argv[2]
output_path = sys.argv[3]

# Simple model training logic
if model_type == 'linear':
    from sklearn.linear_model import LinearRegression
    model = LinearRegression()
elif model_type == 'random_forest':
    from sklearn.ensemble import RandomForestRegressor
    model = RandomForestRegressor()
else:
    from sklearn.linear_model import LogisticRegression
    model = LogisticRegression()

# Dummy training (in real scenario, you'd split features/targets)
X = data.iloc[:, :-1]
y = data.iloc[:, -1]
model.fit(X, y)

# Save model
joblib.dump(model, output_path)
print(f'Model trained and saved to {output_path}')
"""
                    ],
                    inputs=[
                        ComponentInput('training_data', 'Dataset',
                                       'Training dataset'),
                        ComponentInput('model_type', 'String',
                                       'Type of model to train', 'linear'),
                        ComponentInput('model_output_path', 'String',
                                       'Path to save trained model')
                    ],
                    outputs=[
                        ComponentOutput('trained_model', 'Model',
                                        'Trained machine learning model')
                    ],
                    resource_requests={
                        'cpu': '500m',
                        'memory': '1Gi'
                    }),
            'model_evaluator':
                PipelineComponent(
                    name='model-evaluator',
                    description='Evaluate a trained model',
                    image='python:3.9-slim',
                    command=['python'],
                    args=[
                        '-c', """
import pandas as pd
import joblib
import sys
from sklearn.metrics import accuracy_score, mean_squared_error
import json

# Load model and test data
model = joblib.load(sys.argv[1])
test_data = pd.read_csv(sys.argv[2])
metrics_output = sys.argv[3]

# Evaluate model
X_test = test_data.iloc[:, :-1]
y_test = test_data.iloc[:, -1]
predictions = model.predict(X_test)

# Calculate metrics
try:
    accuracy = accuracy_score(y_test, predictions)
    metrics = {'accuracy': accuracy}
except:
    mse = mean_squared_error(y_test, predictions)
    metrics = {'mse': mse}

# Save metrics
with open(metrics_output, 'w') as f:
    json.dump(metrics, f)

print(f'Model evaluation completed: {metrics}')
"""
                    ],
                    inputs=[
                        ComponentInput('model_path', 'Model',
                                       'Path to trained model'),
                        ComponentInput('test_data', 'Dataset',
                                       'Test dataset for evaluation'),
                        ComponentInput('metrics_output_path', 'String',
                                       'Path to save evaluation metrics')
                    ],
                    outputs=[
                        ComponentOutput('evaluation_metrics', 'Metrics',
                                        'Model evaluation metrics')
                    ],
                    resource_requests={
                        'cpu': '200m',
                        'memory': '512Mi'
                    })
        }

    def create_random_component(self, requirements: str) -> PipelineComponent:
        """Create a random component based on requirements."""
        component_id = str(uuid.uuid4())[:8]

        # Parse requirements for hints
        req_lower = requirements.lower()

        # Determine component type and characteristics
        if any(word in req_lower for word in ['data', 'load', 'extract']):
            base_name = 'data-processor'
            description = 'Process and transform data based on requirements'
            image = random.choice(['python:3.9-slim', 'pandas/pandas:latest'])
        elif any(word in req_lower for word in ['train', 'model', 'ml', 'ai']):
            base_name = 'model-component'
            description = 'Machine learning model component'
            image = random.choice([
                'tensorflow/tensorflow:2.13.0',
                'pytorch/pytorch:2.0.1-py3.9-cuda11.7-cudnn8-runtime'
            ])
        elif any(word in req_lower
                 for word in ['visual', 'plot', 'chart', 'graph']):
            base_name = 'visualization'
            description = 'Create visualizations and reports'
            image = 'jupyter/scipy-notebook:latest'
        else:
            base_name = 'custom-component'
            description = f"Custom component for: {requirements}"
            image = random.choice(self.common_images)

        # Generate inputs and outputs
        num_inputs = random.randint(1, 4)
        num_outputs = random.randint(1, 3)

        inputs = []
        for i in range(num_inputs):
            input_types = [
                'String', 'Dataset', 'Model', 'Metrics', 'Integer', 'Float'
            ]
            inputs.append(
                ComponentInput(
                    name=f"input_{i+1}",
                    type=random.choice(input_types),
                    description=f"Input parameter {i+1}"))

        outputs = []
        for i in range(num_outputs):
            output_types = ['Dataset', 'Model', 'Metrics', 'String', 'Artifact']
            outputs.append(
                ComponentOutput(
                    name=f"output_{i+1}",
                    type=random.choice(output_types),
                    description=f"Output result {i+1}"))

        return PipelineComponent(
            name=f"{base_name}-{component_id}",
            description=description,
            image=image,
            command=['python'],
            args=[
                '-c', f"# Component for: {requirements}\nprint('Processing...')"
            ],
            inputs=inputs,
            outputs=outputs,
            resource_requests={
                'cpu': '100m',
                'memory': '256Mi'
            })

    def get_component(self, component_name: str) -> Optional[PipelineComponent]:
        """Get a predefined component by name."""
        return self.predefined_components.get(component_name)

    def list_available_components(self) -> List[str]:
        """List all available predefined components."""
        return list(self.predefined_components.keys())

    def create_pipeline_spec(self, name: str, description: str,
                             components: List[str],
                             requirements: str) -> PipelineSpec:
        """Create a complete pipeline specification."""
        pipeline_components = []
        pipeline_tasks = []

        # Add requested components
        for comp_name in components:
            if comp_name in self.predefined_components:
                pipeline_components.append(
                    self.predefined_components[comp_name])
            else:
                # Create a custom component
                custom_comp = self.create_random_component(
                    f"Custom component: {comp_name}")
                custom_comp.name = comp_name
                pipeline_components.append(custom_comp)

        # Create tasks with dependencies
        for i, component in enumerate(pipeline_components):
            task_inputs = {}
            depends_on = []

            # Set up inputs and dependencies
            for input_param in component.inputs:
                if i == 0:  # First component
                    task_inputs[
                        input_param
                        .name] = f"{{$.inputs.parameters.{input_param.name}}}"
                else:
                    # Connect to previous component's output
                    prev_component = pipeline_components[i - 1]
                    if prev_component.outputs:
                        task_inputs[
                            input_param.
                            name] = f"{{$.tasks.{prev_component.name}.outputs.artifacts.{prev_component.outputs[0].name}}}"
                    depends_on.append(prev_component.name)

            pipeline_tasks.append(
                PipelineTask(
                    name=component.name,
                    component_ref=component.name,
                    inputs=task_inputs,
                    depends_on=depends_on if depends_on else None))

        return PipelineSpec(
            name=name,
            description=description,
            components=pipeline_components,
            tasks=pipeline_tasks)

    def generate_kubeflow_yaml(self, pipeline_spec: PipelineSpec) -> str:
        """Generate Kubeflow Pipeline YAML from pipeline spec."""
        kf_pipeline = {
            'apiVersion': 'argoproj.io/v1alpha1',
            'kind': 'Workflow',
            'metadata': {
                'generateName': f"{pipeline_spec.name}-",
                'annotations': {
                    'pipelines.kubeflow.org/kfp_sdk_version':
                        '2.1.0',
                    'pipelines.kubeflow.org/pipeline_compilation_time':
                        datetime.now().isoformat(),
                    'pipelines.kubeflow.org/pipeline_spec':
                        json.dumps({
                            'description': pipeline_spec.description,
                            'name': pipeline_spec.name
                        })
                }
            },
            'spec': {
                'entrypoint': 'pipeline',
                'templates': []
            }
        }

        # Add component templates
        for component in pipeline_spec.components:
            template = {
                'name': component.name,
                'container': {
                    'image': component.image,
                    'command': component.command,
                    'args': component.args,
                    'resources': {
                        'requests':
                            component.resource_requests or {
                                'cpu': '100m',
                                'memory': '256Mi'
                            }
                    }
                },
                'inputs': {
                    'parameters': [{
                        'name': inp.name,
                        'description': inp.description
                    } for inp in component.inputs]
                },
                'outputs': {
                    'artifacts': [{
                        'name': out.name,
                        'path': f"/tmp/{out.name}"
                    } for out in component.outputs]
                }
            }
            kf_pipeline['spec']['templates'].append(template)

        # Add main pipeline template
        dag_tasks = []
        for task in pipeline_spec.tasks:
            dag_task = {
                'name': task.name,
                'template': task.component_ref,
                'arguments': {
                    'parameters': [{
                        'name': k,
                        'value': v
                    } for k, v in task.inputs.items()]
                }
            }
            if task.depends_on:
                dag_task['dependencies'] = task.depends_on

            dag_tasks.append(dag_task)

        pipeline_template = {'name': 'pipeline', 'dag': {'tasks': dag_tasks}}
        kf_pipeline['spec']['templates'].append(pipeline_template)

        return yaml.dump(kf_pipeline, default_flow_style=False, sort_keys=False)


# Initialize the MCP server
server = Server('pipeline-generator')
generator = PipelineGenerator()


@server.list_tools()
async def handle_list_tools() -> List[Tool]:
    """List available tools for pipeline generation."""
    return [
        Tool(
            name='create_component',
            description='Create a pipeline component (specific or random) based on requirements',
            inputSchema={
                'type': 'object',
                'properties': {
                    'requirements': {
                        'type':
                            'string',
                        'description':
                            "Requirements for the component (e.g., 'data loading', 'model training', 'visualization')"
                    },
                    'component_name': {
                        'type':
                            'string',
                        'description':
                            'Specific component name (optional). If provided, will use predefined component.'
                    },
                    'random': {
                        'type': 'boolean',
                        'description': 'Whether to create a random component',
                        'default': False
                    }
                },
                'required': ['requirements']
            }),
        Tool(
            name='list_components',
            description='List all available predefined components',
            inputSchema={
                'type': 'object',
                'properties': {},
                'required': []
            }),
        Tool(
            name='create_pipeline_spec',
            description='Create a complete pipeline specification',
            inputSchema={
                'type':
                    'object',
                'properties': {
                    'name': {
                        'type': 'string',
                        'description': 'Name of the pipeline'
                    },
                    'description': {
                        'type': 'string',
                        'description': 'Description of the pipeline'
                    },
                    'components': {
                        'type':
                            'array',
                        'items': {
                            'type': 'string'
                        },
                        'description':
                            'List of component names to include in the pipeline'
                    },
                    'requirements': {
                        'type':
                            'string',
                        'description':
                            'Overall requirements and goals for the pipeline'
                    }
                },
                'required': [
                    'name', 'description', 'components', 'requirements'
                ]
            }),
        Tool(
            name='generate_pipeline_yaml',
            description='Generate Kubeflow Pipeline YAML from a pipeline specification',
            inputSchema={
                'type': 'object',
                'properties': {
                    'pipeline_spec': {
                        'type':
                            'object',
                        'description':
                            'Pipeline specification object (from create_pipeline_spec)'
                    }
                },
                'required': ['pipeline_spec']
            }),
        Tool(
            name='create_complete_pipeline',
            description='Create a complete pipeline from requirements (combines all steps)',
            inputSchema={
                'type': 'object',
                'properties': {
                    'name': {
                        'type': 'string',
                        'description': 'Name of the pipeline'
                    },
                    'description': {
                        'type': 'string',
                        'description': 'Description of the pipeline'
                    },
                    'requirements': {
                        'type': 'string',
                        'description': 'Detailed requirements for the pipeline'
                    },
                    'num_components': {
                        'type':
                            'integer',
                        'description':
                            'Number of components to include (default: 3)',
                        'default':
                            3
                    }
                },
                'required': ['name', 'description', 'requirements']
            })
    ]


@server.call_tool()
async def handle_call_tool(name: str, arguments: dict) -> CallToolResult:
    """Handle tool calls for pipeline generation."""

    try:
        if name == 'create_component':
            requirements = arguments.get('requirements', '')
            component_name = arguments.get('component_name')
            is_random = arguments.get('random', False)

            if component_name and not is_random:
                # Try to get predefined component
                component = generator.get_component(component_name)
                if component:
                    result = asdict(component)
                    result['source'] = 'predefined'
                else:
                    # Create custom component with the specified name
                    component = generator.create_random_component(requirements)
                    component.name = component_name
                    result = asdict(component)
                    result['source'] = 'custom'
            else:
                # Create random component
                component = generator.create_random_component(requirements)
                result = asdict(component)
                result['source'] = 'random'

            return CallToolResult(content=[
                TextContent(
                    type='text',
                    text=f"Created component:\n\n```json\n{json.dumps(result, indent=2)}\n```"
                )
            ])

        elif name == 'list_components':
            components = generator.list_available_components()
            component_details = {}

            for comp_name in components:
                comp = generator.get_component(comp_name)
                component_details[comp_name] = {
                    'description': comp.description,
                    'inputs': len(comp.inputs),
                    'outputs': len(comp.outputs)
                }

            return CallToolResult(content=[
                TextContent(
                    type='text',
                    text=f"Available predefined components:\n\n```json\n{json.dumps(component_details, indent=2)}\n```"
                )
            ])

        elif name == 'create_pipeline_spec':
            name = arguments.get('name', '')
            description = arguments.get('description', '')
            components = arguments.get('../components', [])
            requirements = arguments.get('requirements', '')

            pipeline_spec = generator.create_pipeline_spec(
                name, description, components, requirements)
            result = asdict(pipeline_spec)

            return CallToolResult(content=[
                TextContent(
                    type='text',
                    text=f"Created pipeline specification:\n\n```json\n{json.dumps(result, indent=2)}\n```"
                )
            ])

        elif name == 'generate_pipeline_yaml':
            pipeline_spec_dict = arguments.get('pipeline_spec', {})

            # Convert dict back to PipelineSpec object
            components = [
                PipelineComponent(**comp)
                for comp in pipeline_spec_dict.get('components', [])
            ]
            tasks = [
                PipelineTask(**task)
                for task in pipeline_spec_dict.get('tasks', [])
            ]

            pipeline_spec = PipelineSpec(
                name=pipeline_spec_dict.get('name', ''),
                description=pipeline_spec_dict.get('description', ''),
                components=components,
                tasks=tasks,
                pipeline_spec_version=pipeline_spec_dict.get(
                    'pipeline_spec_version', '2.1.0'))

            yaml_content = generator.generate_kubeflow_yaml(pipeline_spec)

            return CallToolResult(content=[
                TextContent(
                    type='text',
                    text=f"Generated Kubeflow Pipeline YAML:\n\n```yaml\n{yaml_content}\n```"
                )
            ])

        elif name == 'create_complete_pipeline':
            name = arguments.get('name', '')
            description = arguments.get('description', '')
            requirements = arguments.get('requirements', '')
            num_components = arguments.get('num_components', 3)

            # Determine component types based on requirements
            req_lower = requirements.lower()
            components = []

            if any(word in req_lower for word in ['data', 'load', 'extract']):
                components.append('data_loader')
            if any(word in req_lower
                   for word in ['preprocess', 'clean', 'transform']):
                components.append('data_preprocessor')
            if any(word in req_lower for word in ['train', 'model', 'ml']):
                components.append('model_trainer')
            if any(word in req_lower for word in ['eval', 'test', 'validate']):
                components.append('model_evaluator')

            # Fill remaining slots with custom components
            while len(components) < num_components:
                custom_name = f"custom_component_{len(components)+1}"
                components.append(custom_name)

            # Create pipeline spec
            pipeline_spec = generator.create_pipeline_spec(
                name, description, components, requirements)

            # Generate YAML
            yaml_content = generator.generate_kubeflow_yaml(pipeline_spec)

            result = {
                'pipeline_spec': asdict(pipeline_spec),
                'kubeflow_yaml': yaml_content
            }

            return CallToolResult(content=[
                TextContent(
                    type='text',
                    text=f"Created complete pipeline:\n\n**Pipeline Specification:**\n```json\n{json.dumps(asdict(pipeline_spec), indent=2)}\n```\n\n**Kubeflow YAML:**\n```yaml\n{yaml_content}\n```"
                )
            ])

        else:
            return CallToolResult(
                content=[
                    TextContent(type='text', text=f"Unknown tool: {name}")
                ],
                isError=True)

    except Exception as e:
        logger.error(f"Error in tool {name}: {str(e)}")
        return CallToolResult(
            content=[
                TextContent(
                    type='text', text=f"Error executing tool {name}: {str(e)}")
            ],
            isError=True)


async def main():
    """Main function to run the MCP server."""
    from mcp.server.stdio import stdio_server

    async with stdio_server() as (read_stream, write_stream):
        await server.run(
            read_stream, write_stream,
            InitializationOptions(
                server_name='pipeline-generator',
                server_version='1.0.0',
                capabilities=server.get_capabilities(
                    notification_options=NotificationOptions(),
                    experimental_capabilities={})))


if __name__ == '__main__':
    import asyncio
    asyncio.run(main())
