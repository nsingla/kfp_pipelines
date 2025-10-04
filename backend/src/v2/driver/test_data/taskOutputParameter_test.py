import functools

from kubernetes import client
import kfp
from kfp import dsl
from kfp.dsl import (
    Input,
    Output,
    Artifact,
    Dataset,
    component, OutputPath
)

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)

@component
def create_dataset(output_parameter_path: OutputPath(str)):
    with open(output_parameter_path, "w") as f:
        f.write('hurricane')

@component
def process_dataset(input_dataset: str, output_int: OutputPath(int)):
    assert input_dataset == "hurricane"
    with open(output_int, "w") as f:
        f.write("100")

@component
def analyze_artifact(data_input: int, output_opinion: OutputPath(bool)):
    assert data_input == 100
    with open(output_opinion, "w") as f:
        f.write(True)

@dsl.pipeline
def primary_pipeline():
    create_dataset_task = create_dataset()
    processed_task = process_dataset(input_dataset=create_dataset_task.outputs["output_parameter_path"])
    analyze_artifact(data_input=processed_task.outputs["output_int"])

if __name__ == '__main__':
    from kfp import compiler

    compiler.Compiler().compile(
        pipeline_func=primary_pipeline,
        package_path=__file__+".yaml"
    )