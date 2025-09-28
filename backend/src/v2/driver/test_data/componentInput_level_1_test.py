import functools

from kfp import dsl
from kfp.dsl import Input, Output, Dataset, Model, Artifact

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)


@dsl.component
def process_inputs(
        name: str,
        number: int,
        threshold: float,
        active: bool,
        a_runtime_var: str,
        output_text: Output[Dataset]
) -> None:
    with open(output_text.path, 'w') as f:
        f.write(f"Name: {name}\n")
        f.write(f"Number doubled: {number * 2}\n")
        f.write(f"Threshold plus 10: {threshold + 10.0}\n")
        f.write(f"Active status: {'enabled' if active else 'disabled'}\n")

    assert name == "default_name"
    assert number == 42
    assert threshold == 0.5
    assert active == True
    assert a_runtime_var == "foo"

@dsl.pipeline
def primary_pipeline(
        name: str = "default_name",
        number: int = 42,
        threshold: float = 0.5,
        active: bool = True,
):
    process_inputs(
        name=name,
        number=number,
        threshold=threshold,
        active=active,
        a_runtime_var="foo",
    )

if __name__ == '__main__':
    from kfp import compiler

    compiler.Compiler().compile(
        pipeline_func=primary_pipeline,
        package_path=__file__+".yaml"
    )
