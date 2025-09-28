import functools

from kfp import dsl, compiler

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)

@dsl.component()
def output_msg(output_artifact: dsl.Output[dsl.Artifact], a_msg: dsl.OutputPath(str)):
    with open(output_artifact.path, 'w') as f:
        f.write("some_artifact")
    with open(a_msg, 'w') as f:
        f.write("this")

@dsl.component()
def print_msg(input_artifact: dsl.Input[dsl.Artifact], msg: str):
    artifact = ""
    with open(input_artifact.path, 'r') as f:
        artifact = f.read()

    assert artifact == "some_artifact"
    assert msg == "this"

@dsl.pipeline
def primary_pipeline():
    output = output_msg()
    with dsl.If('this' == output.outputs['a_msg']):
        print_msg(input_artifact=output.outputs['output_artifact'], msg="this")
    with dsl.Else():
        print_msg(input_artifact=output.outputs['output_artifact'], msg="that")


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=primary_pipeline,
        package_path=__file__ + ".yaml"
    )
