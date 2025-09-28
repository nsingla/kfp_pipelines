import functools

import kfp
from kfp import dsl
from kfp.dsl import (
    Input,
    Output,
    Artifact,
    Dataset,
    component,
    pipeline
)

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)

@component
def create_dataset(output_dataset: Output[Dataset]):
    with open(output_dataset.path, "w") as f:
        f.write('hurricane')
    output_dataset.metadata["category"] = 5
    output_dataset.metadata["description"] = "A simple dataset with hurricane"

@component
def process_dataset(input_dataset: Input[Dataset], output_artifact: Output[Artifact]):
    with open(input_dataset.path, "r") as f:
        data = f.read()
    with open(output_artifact.path, "w") as f:
        f.write("very_bad")

@component
def analyze_artifact(artifact: Input[Artifact], output_artifact: Output[Artifact]):
    with open(artifact.path, "r") as f:
        data = f.read()
    assert data == "very_bad"
    with open(output_artifact.path, "w") as f:
        f.write(f'done_analyzing')

@component
def evaluate_analysis(analysis_artifact: Input[Artifact]) -> str:
    with open(analysis_artifact.path, "r") as f:
        data = f.read()
    assert data == "done_analyzing"
    return "good"

@pipeline
def secondary_pipeline(input_dataset: Input[Dataset]) -> Artifact:
    processed = process_dataset(input_dataset=input_dataset)
    analysis = analyze_artifact(artifact=processed.outputs["output_artifact"])
    return analysis.outputs["output_artifact"]


@pipeline(name="artifact-passing-example")
def primary_pipeline():
    dataset_op = create_dataset()
    nested_pipeline_op = secondary_pipeline(input_dataset=dataset_op.outputs["output_dataset"])
    evaluate_analysis(analysis_artifact=nested_pipeline_op.output)

if __name__ == '__main__':
    from kfp import compiler

    compiler.Compiler().compile(
        pipeline_func=primary_pipeline,
        package_path=__file__+".yaml"
    )