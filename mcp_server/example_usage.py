#!/usr/bin/env python3
"""Example usage of the MCP Pipeline Server.

This script demonstrates how to interact with the MCP server to generate
pipeline components and complete pipelines.
"""

import json

from mcp_pipeline_server import PipelineGenerator


def example_component_creation():
    """Example of creating individual components."""
    print('=== Component Creation Examples ===\n')

    generator = PipelineGenerator()

    # Example 1: Get a predefined component
    print('1. Getting predefined data loader component:')
    data_loader = generator.get_component('data_loader')
    if data_loader:
        print(f"   Name: {data_loader.name}")
        print(f"   Description: {data_loader.description}")
        print(f"   Image: {data_loader.image}")
        print(f"   Inputs: {len(data_loader.inputs)}")
        print(f"   Outputs: {len(data_loader.outputs)}")

    print('\n' + '=' * 50 + '\n')

    # Example 2: Create a random component
    print('2. Creating random component for data visualization:')
    viz_component = generator.create_random_component(
        'Create charts and visualizations for data analysis')
    print(f"   Name: {viz_component.name}")
    print(f"   Description: {viz_component.description}")
    print(f"   Image: {viz_component.image}")
    print(f"   Inputs: {[inp.name for inp in viz_component.inputs]}")
    print(f"   Outputs: {[out.name for out in viz_component.outputs]}")

    print('\n' + '=' * 50 + '\n')

    # Example 3: List all available components
    print('3. Available predefined components:')
    components = generator.list_available_components()
    for comp_name in components:
        comp = generator.get_component(comp_name)
        print(f"   - {comp_name}: {comp.description}")


def example_pipeline_creation():
    """Example of creating complete pipelines."""
    print('=== Pipeline Creation Examples ===\n')

    generator = PipelineGenerator()

    # Example 1: Create a simple ML pipeline
    print('1. Creating ML training pipeline:')
    pipeline_spec = generator.create_pipeline_spec(
        name='fraud-detection-ml',
        description='Machine learning pipeline for fraud detection',
        components=[
            'data_loader', 'data_preprocessor', 'model_trainer',
            'model_evaluator'
        ],
        requirements='Train a fraud detection model using transaction data')

    print(f"   Pipeline Name: {pipeline_spec.name}")
    print(f"   Description: {pipeline_spec.description}")
    print(f"   Number of Components: {len(pipeline_spec.components)}")
    print(f"   Number of Tasks: {len(pipeline_spec.tasks)}")

    # Show component flow
    print('   Component Flow:')
    for i, task in enumerate(pipeline_spec.tasks):
        deps = f" (depends on: {task.depends_on})" if task.depends_on else ''
        print(f"     {i+1}. {task.name}{deps}")

    print('\n' + '=' * 50 + '\n')

    # Example 2: Generate Kubeflow YAML
    print('2. Generating Kubeflow YAML:')
    yaml_content = generator.generate_kubeflow_yaml(pipeline_spec)
    print('   YAML generated successfully!')
    print(f"   YAML length: {len(yaml_content)} characters")

    # Save to file
    with open('example_pipeline.yaml', 'w') as f:
        f.write(yaml_content)
    print('   Saved to: example_pipeline.yaml')


def example_smart_pipeline_generation():
    """Example of smart pipeline generation based on requirements."""
    print('=== Smart Pipeline Generation ===\n')

    generator = PipelineGenerator()

    # Example scenarios with different requirements
    scenarios = [{
        'name':
            'image-classification',
        'description':
            'Image classification pipeline',
        'requirements':
            'Load image data, preprocess images, train a CNN model, and evaluate accuracy'
    }, {
        'name':
            'time-series-forecasting',
        'description':
            'Time series forecasting pipeline',
        'requirements':
            'Load time series data, clean and transform data, train forecasting model, validate predictions'
    }, {
        'name':
            'nlp-sentiment-analysis',
        'description':
            'NLP sentiment analysis pipeline',
        'requirements':
            'Load text data, preprocess text, train sentiment model, evaluate performance'
    }]

    for i, scenario in enumerate(scenarios, 1):
        print(f"{i}. Scenario: {scenario['name']}")
        print(f"   Requirements: {scenario['requirements']}")

        # Determine components based on requirements
        req_lower = scenario['requirements'].lower()
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

        # Add custom components for specific domains
        if 'image' in req_lower:
            components.append('image_preprocessor')
        elif 'time' in req_lower:
            components.append('time_series_processor')
        elif 'nlp' in req_lower or 'text' in req_lower:
            components.append('text_preprocessor')

        print(f"   Selected Components: {components}")

        # Create pipeline spec
        pipeline_spec = generator.create_pipeline_spec(
            name=scenario['name'],
            description=scenario['description'],
            components=components,
            requirements=scenario['requirements'])

        print(
            f"   Generated Pipeline: {len(pipeline_spec.components)} components, {len(pipeline_spec.tasks)} tasks"
        )
        print()


def example_json_output():
    """Example of generating JSON output for MCP responses."""
    print('=== JSON Output Examples ===\n')

    generator = PipelineGenerator()

    # Create a component and show JSON representation
    component = generator.create_random_component(
        'Process financial data for risk analysis')

    # Convert to dict (as would be done in MCP server)
    from dataclasses import asdict
    component_dict = asdict(component)
    component_dict['source'] = 'random'

    print('1. Component JSON Output:')
    print(json.dumps(component_dict, indent=2))

    print('\n' + '=' * 50 + '\n')

    # Create a pipeline spec and show JSON
    pipeline_spec = generator.create_pipeline_spec(
        name='risk-analysis-pipeline',
        description='Financial risk analysis pipeline',
        components=['data_loader', 'financial_processor', 'risk_model'],
        requirements='Analyze financial data to assess risk levels')

    pipeline_dict = asdict(pipeline_spec)

    print('2. Pipeline Specification JSON Output:')
    print(json.dumps(pipeline_dict, indent=2)[:500] +
          '...')  # Truncated for readability


def main():
    """Run all examples."""
    print('MCP Pipeline Server - Usage Examples')
    print('=' * 60)
    print()

    try:
        example_component_creation()
        print('\n' + '=' * 60 + '\n')

        example_pipeline_creation()
        print('\n' + '=' * 60 + '\n')

        example_smart_pipeline_generation()
        print('\n' + '=' * 60 + '\n')

        example_json_output()

        print('\n' + '=' * 60)
        print('Examples completed successfully!')
        print('Check the generated files:')
        print('- example_pipeline.yaml')

    except Exception as e:
        print(f"Error running examples: {e}")
        import traceback
        traceback.print_exc()


if __name__ == '__main__':
    main()
