import functools
from typing import List

from kfp import dsl
from kfp.dsl import (
    Input,
    Output,
    Artifact,
    Dataset,
    component, pipeline, ParallelFor, Collected
)

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)


@component()
def split_ids(model_ids: str) -> list:
    return model_ids.split(',')

@component()
def create_file(file: Output[Artifact], content: str):
    print(f'Creating file with content: {content}')
    with open(file.path, 'w') as f:
        f.write(content)

@component()
def read_files(files: List[Artifact]) -> str:
    collect = []
    for f in files:
        print(f'Reading artifact {f.name} file: {f.path}')
        with open(f.path, 'r') as f:
            data = f.read()
            print(data)
            collect.append(data)
    print(collect)
    assert collect == ['s1', 's2', 's3']
    return 'files read'


@component()
def read_single_file(file: Artifact, expected: str) -> str:
    print(f'Reading file: {file.path}')
    with open(file.path, 'r') as f:
        data = f.read()
        print(data)
        assert expected == data
    return file.uri

@pipeline()
def secondary_pipeline(model_ids: str = '',) -> List[Artifact]:
    ids_split_op = split_ids(model_ids=model_ids)
    with ParallelFor(ids_split_op.output) as model_id:
        create_file_op = create_file(content=model_id)
        read_single_file(file=create_file_op.outputs['file'], expected=model_id)
    read_files(files=Collected(create_file_op.outputs['file']))
    return Collected(create_file_op.outputs['file'])


@pipeline()
def primary_pipeline():
    model_ids = 's1,s2,s3'
    dag = secondary_pipeline(model_ids=model_ids)
    read_files(files=dag.output)

if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=primary_pipeline,
        package_path=__file__ + ".yaml"
    )
