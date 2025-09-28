import functools

from kfp import dsl
from kfp.dsl import (
    Input,
    Output,
    Artifact,
    Dataset,
    component
)

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)

@component
def create_dataset(output_dataset: Output[Dataset], condition_result: dsl.OutputPath(str)):
    with open(output_dataset.path, "w") as f:
        f.write('{"values": [1, 2, 3, 4, 5]}')
    output_dataset.metadata["item_count"] = 5
    output_dataset.metadata["description"] = "A simple dataset with integers"
    with open(condition_result, 'w') as f:
        f.write("this")

@component
def process_dataset(some_param: str, input_dataset: Input[Dataset], output_artifact: Output[Artifact]):
    with open(input_dataset.path, "r") as f:
        data = f.read()
    with open(output_artifact.path, "w") as f:
        f.write(f'{{"values": {data}}}')

@component
def analyze_artifact(analyze_artifact_input: Input[Artifact], analyze_output_artifact: Output[Artifact]):
    with open(analyze_artifact_input.path, "r") as f:
        data = f.read()
    with open(analyze_output_artifact.path, "w") as f:
        f.write(f'{{"values": {data}}}')


@dsl.pipeline
def primary_pipeline():
    create_dataset_task = create_dataset()
    with dsl.If('this' == create_dataset_task.outputs['condition_result']):
        process_dataset_task = process_dataset(some_param="this", input_dataset=create_dataset_task.outputs['output_dataset'])
        analyze_artifact(analyze_artifact_input=process_dataset_task.outputs["output_artifact"])
    with dsl.Else():
        process_dataset(some_param="that", input_dataset=create_dataset_task.outputs['output_dataset'])

if __name__ == '__main__':
    from kfp import compiler

    compiler.Compiler().compile(
        pipeline_func=primary_pipeline,
        package_path=__file__+".yaml"
    )