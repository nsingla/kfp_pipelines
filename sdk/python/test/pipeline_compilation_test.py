import os.path

import pytest
import re
import tempfile
from dataclasses import dataclass
from typing import Optional, Callable

import kfp
from kfp import compiler
from sdk.python.test.test_utils.comparison_utils import ComparisonUtils
from sdk.python.test.test_utils.file_utils import FileUtils

from test_data.components.add_numbers import add_numbers
from test_data.components.hello_world import echo
from test_data.components.two_step_pipeline import my_pipeline as two_step_pipeline
from test_data.components.pipeline_with_condition import my_pipeline as condition_pipeline
from test_data.components.pipeline_with_loops import my_pipeline as loops_pipeline
from test_data.components.pipeline_with_outputs import my_pipeline as outputs_pipeline
from test_data.components.collected_parameters import collected_param_pipeline
from test_data.components.component_with_optional_inputs import pipeline
from test_data.components.mixed_parameters import crust as mixed_parameters_pipeline
from test_data.components.producer_consumer_param import producer_consumer_param_pipeline
from test_data.components.parameter_cache import crust as parameter_cache_pipeline
from test_data.components.parameter_oneof import crust as parameter_oneof_pipeline
from test_data.components.pipeline_with_env import my_pipeline as env_pipeline
from test_data.components.pipeline_with_retry import my_pipeline as retry_pipeline
from test_data.components.parallel_after_dependency import loop_with_after_dependency_set
from test_data.components.multiple_parameters_namedtuple import crust as multiple_params_namedtuple_pipeline
from test_data.components.with_artifacts.multiple_artifacts_namedtuple import crust as multiple_artifacts_namedtuple_pipeline
from test_data.components.modelcar.modelcar import pipeline_modelcar_test
from test_data.components.with_artifacts.artifact_cache import crust as artifact_cache_pipeline
from test_data.components.with_artifacts.artifact_crust import crust as artifact_crust_pipeline
from test_data.components.container_component_with_no_inputs import pipeline as container_no_inputs_pipeline
from test_data.components.loop_consume_upstream import loop_consume_upstream
from test_data.components.parallelfor_fan_in.parameters_simple import math_pipeline as parameters_simple_pipeline
from test_data.components.with_artifacts.pipeline_with_artifact_upload_download import my_pipeline as artifact_upload_download_pipeline
from test_data.components.pipeline_with_input_status_state import status_state_pipeline
from test_data.components.pipeline_with_placeholders import pipeline_with_placeholders
from test_data.components.pipeline_with_k8s_spec.pipeline_with_pod_metadata import pipeline_with_pod_metadata
from test_data.components.pipeline_with_k8s_spec.pipeline_with_secret_as_env import pipeline_secret_env
from test_data.components.pipeline_with_k8s_spec.pipeline_with_workspace import pipeline_with_workspace
from test_data.components.two_step_pipeline_containerized import my_pipeline as two_step_containerized_pipeline
from test_data.components.nested_pipelines.nested_pipeline_opt_input_child_level import nested_pipeline_opt_input_child_level
from test_data.components.pythonic_artifcats.pythonic_artifacts_test_pipeline import pythonic_artifacts_test_pipeline
from test_data.components.components_with_optional_artifacts import pipeline as optional_artifacts_pipeline
from test_data.components.lightweight_python_functions_pipeline import pipeline as lightweight_python_pipeline
from test_data.components.xgboost_sample_pipeline import xgboost_pipeline
from test_data.components.pipeline_with_after import my_pipeline as after_pipeline
from test_data.components.metrics_visualization_v2 import metrics_visualization_pipeline
from test_data.components.pipeline_with_nested_conditions import my_pipeline as nested_conditions_pipeline
from test_data.components.container_io import container_io
from test_data.components.pipeline_with_exit_handler import my_pipeline as exit_handler_pipeline
from test_data.components.pipeline_with_importer import my_pipeline as importer_pipeline
from test_data.components.pipeline_with_nested_loops import my_pipeline as nested_loops_pipeline
from test_data.components.concat_message import concat_message
from test_data.components.preprocess import preprocess
from test_data.components.sequential_v2 import sequential
from test_data.components.parallelfor_fan_in.artifacts_simple import math_pipeline as artifacts_simple_pipeline
from test_data.components.pipeline_with_multiple_exit_handlers import my_pipeline as multiple_exit_handlers_pipeline
from test_data.components.pipeline_with_reused_component import my_pipeline as reused_component_pipeline
from test_data.components.parallelfor_fan_in.artifacts_complex import math_pipeline as artifacts_complex_pipeline
from test_data.components.parallelfor_fan_in.conditional_producer_and_consumers import math_pipeline as conditional_producer_consumers_pipeline
from test_data.components.with_artifacts.collected_artifacts import collected_artifact_pipeline
from test_data.components.pythonic_artifcats.pythonic_artifacts_with_multiple_returns import split_datasets_and_return_first as pythonic_artifacts_multiple_returns
from test_data.components.identity import identity
from test_data.components.input_artifact import input_artifact
from test_data.components.nested_return import nested_return
from test_data.components.pipeline_in_pipeline import my_pipeline as pipeline_in_pipeline
from test_data.components.container_with_concat_placeholder import container_with_concat_placeholder
from test_data.components.lightweight_python_functions_with_outputs import pipeline as lightweight_python_with_outputs_pipeline
from test_data.components.output_metrics import output_metrics
from test_data.components.dict_input import dict_input
from test_data.components.pipeline_with_k8s_spec.pipeline_with_volume import pipeline_with_volume
from test_data.components.pipeline_with_k8s_spec.pipeline_with_volume_no_cache import pipeline_with_volume_no_cache
from test_data.components.container_with_if_placeholder import container_with_if_placeholder
from test_data.components.container_with_placeholder_in_fstring import container_with_placeholder_in_fstring
from test_data.components.pipeline_in_pipeline_complex import my_pipeline as pipeline_in_pipeline_complex
from test_data.components.if_elif_else_complex import lucky_number_pipeline
from test_data.components.if_elif_else_with_oneof_parameters import outer_pipeline as if_elif_else_oneof_params_pipeline
from test_data.components.if_else_with_oneof_parameters import flip_coin_pipeline as if_else_oneof_params_pipeline
from test_data.components.if_else_with_oneof_artifacts import outer_pipeline as if_else_oneof_artifacts_pipeline
from test_data.components.pipeline_with_loops_and_conditions import my_pipeline as loops_and_conditions_pipeline
from test_data.components.pipeline_with_various_io_types import my_pipeline as various_io_types_pipeline
from test_data.components.pipeline_with_metrics_outputs import my_pipeline as metrics_outputs_pipeline
from test_data.components.containerized_python_component import concat_message as containerized_concat_message
from test_data.components.with_artifacts.container_with_artifact_output import container_with_artifact_output
from test_data.components.parallelfor_fan_in.nested_with_parameters import math_pipeline as nested_with_parameters_pipeline
from test_data.components.parallelfor_fan_in.parameters_complex import math_pipeline as parameters_complex_pipeline
# Additional missing pipeline imports
from test_data.components.component_with_metadata_fields import dataset_joiner
from test_data.components.with_pip_customizations.component_with_pip_index_urls import pipeline as pip_index_urls_pipeline
from test_data.components.with_pip_customizations.component_with_pip_install_in_venv import component_with_pip_install as pip_install_venv_pipeline
from test_data.components.with_pip_customizations.component_with_pip_install import component_with_pip_install as pip_install_pipeline
from test_data.components.component_with_task_final_status import exit_comp as task_final_status_pipeline
from test_data.components.parallelfor_fan_in.pipeline_producer_consumer import math_pipeline as producer_consumer_parallel_for_pipeline
from test_data.components.pipeline_with_utils import pipeline_with_utils
from test_data.components.pipeline_with_k8s_spec.pipeline_with_secret_as_volume import pipeline_secret_volume
from test_data.components.flip_coin import flipcoin_pipeline as flip_coin

# Additional missing pipeline imports for remaining files
from test_data.components.arguments_parameters import echo as arguments_parameters_echo
from test_data.components.container_no_input import container_no_input
from test_data.components.env_var import test_env_exists
from test_data.components.pipeline_with_k8s_spec.create_pod_metadata_complex import pipeline_with_pod_metadata as create_pod_metadata_complex
from test_data.components.nested_pipelines.nested_pipeline_opt_inputs_nil import nested_pipeline_opt_inputs_nil
from test_data.components.nested_pipelines.nested_pipeline_opt_inputs_parent_level import nested_pipeline_opt_inputs_parent_level

# Additional remaining missing pipeline imports
from test_data.components.cross_loop_after_topology import my_pipeline as cross_loop_after_topology_pipeline
from test_data.components.pipeline_as_exit_task import my_pipeline as pipeline_as_exit_task
from test_data.components.pipeline_in_pipeline_loaded_from_yaml import my_pipeline as pipeline_in_pipeline_loaded_from_yaml
from test_data.components.pipeline_with_dynamic_importer_metadata import my_pipeline as pipeline_with_dynamic_importer_metadata
from test_data.components.pipeline_with_google_artifact_type import my_pipeline as pipeline_with_google_artifact_type
from test_data.components.pipeline_with_metadata_fields import dataset_concatenator as pipeline_with_metadata_fields
from test_data.components.pythonic_artifcats.pythonic_artifacts_with_list_of_artifacts import make_and_join_datasets as pythonic_artifacts_with_list_of_artifacts
from test_data.components.pythonic_artifcats.pythonic_artifact_with_single_return import make_language_model_pipeline as pythonic_artifact_with_single_return

import yaml


class TestPipelineCompilation:
    _VALID_PIPELINE_FILES = FileUtils.VALID_PIPELINE_FILES

    @dataclass
    class TestData:
        pipeline_name: str
        pipeline_display_name: Optional[str]
        pipeline_func: Callable
        pipline_func_args: Optional[dict]
        compiled_file_name: str
        expected_compiled_file_path: str

        def __str__(self) -> str:
            return (f"Compilation Data: name={self.pipeline_name} "
                    f"compiled_file_name={self.compiled_file_name} "
                    f"expected_file={self.expected_compiled_file_path}")

        def __repr__(self) -> str:
            return self.__str__()


    keys_to_ignore_for_comparison = ['displayName', 'name', 'sdkVersion']

    @pytest.mark.parametrize(
        'pipeline_data',
        [
            TestData(pipeline_display_name='Add Numbers',
                     pipeline_name='add-numbers',
                     pipeline_func=add_numbers,
                     pipline_func_args=None,
                     compiled_file_name='add_numbers.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/add_numbers.yaml'
                     ),
            TestData(pipeline_display_name='Hello World',
                     pipeline_name='hello-world',
                     pipeline_func=echo,
                     pipline_func_args=None,
                     compiled_file_name='hello_world.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/hello-world.yaml'
                     ),
            TestData(pipeline_display_name='Simple Two Step Pipeline',
                     pipeline_name='simple-two-step-pipeline',
                     pipeline_func=two_step_pipeline,
                     pipline_func_args={'text': 'Hello KFP!'},
                     compiled_file_name='two_step_pipeline.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/two_step_pipeline.yaml'
                     ),
            TestData(pipeline_display_name='Single Condition Pipeline',
                     pipeline_name='single-condition-pipeline',
                     pipeline_func=condition_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='condition_pipeline.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_condition.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Loops',
                     pipeline_name='pipeline-with-loops',
                     pipeline_func=loops_pipeline,
                     pipline_func_args={'loop_parameter': ['item1', 'item2']},
                     compiled_file_name='loops_pipeline.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_loops.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Outputs',
                     pipeline_name='pipeline-with-outputs',
                     pipeline_func=outputs_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='outputs_pipeline.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_outputs.yaml'
                     ),
            # Critical test cases
            TestData(pipeline_display_name='Collected Parameters Pipeline',
                     pipeline_name='collected-param-pipeline',
                     pipeline_func=collected_param_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='collected_parameters.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/collected_parameters.yaml'
                     ),
            TestData(pipeline_display_name='Component with Optional Inputs',
                     pipeline_name='component-optional-input',
                     pipeline_func=pipeline,
                     pipline_func_args=None,
                     compiled_file_name='component_with_optional_inputs.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/component_with_optional_inputs.yaml'
                     ),
            TestData(pipeline_display_name='Mixed Parameters Pipeline',
                     pipeline_name='mixed_parameters-pipeline',
                     pipeline_func=mixed_parameters_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='mixed_parameters.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/mixed_parameters.yaml'
                     ),
            TestData(pipeline_display_name='Producer Consumer Param Pipeline',
                     pipeline_name='producer-consumer-param-pipeline',
                     pipeline_func=producer_consumer_param_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='producer_consumer_param_pipeline.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/producer_consumer_param_pipeline.yaml'
                     ),
            TestData(pipeline_display_name='Parameter Cache Pipeline',
                     pipeline_name='parameter_cache-pipeline',
                     pipeline_func=parameter_cache_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='parameter_cache.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/parameter_cache.yaml'
                     ),
            TestData(pipeline_display_name='Parameter OneOf Pipeline',
                     pipeline_name='parameter_oneof-pipeline',
                     pipeline_func=parameter_oneof_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='parameter_oneof.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/parameter_oneof.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Environment Variables',
                     pipeline_name='pipeline-with-env',
                     pipeline_func=env_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_env.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_env.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Retry',
                     pipeline_name='test-pipeline',
                     pipeline_func=retry_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_retry.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_retry.yaml'
                     ),
            TestData(pipeline_display_name='Parallel After Dependency',
                     pipeline_name='loop-with-after-dependency-set',
                     pipeline_func=loop_with_after_dependency_set,
                     pipline_func_args=None,
                     compiled_file_name='parallel_for_after_dependency.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/parallel_for_after_dependency.yaml'
                     ),
            TestData(pipeline_display_name='Multiple Parameters NamedTuple',
                     pipeline_name='multiple_parameters_namedtuple-pipeline',
                     pipeline_func=multiple_params_namedtuple_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='multiple_parameters_namedtuple.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/multiple_parameters_namedtuple.yaml'
                     ),
            TestData(pipeline_display_name='Multiple Artifacts NamedTuple',
                     pipeline_name='multiple_artifacts_namedtuple-pipeline',
                     pipeline_func=multiple_artifacts_namedtuple_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='multiple_artifacts_namedtuple.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/multiple_artifacts_namedtuple.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Modelcar',
                     pipeline_name='pipeline-with-modelcar-model',
                     pipeline_func=pipeline_modelcar_test,
                     pipline_func_args=None,
                     compiled_file_name='modelcar.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/modelcar.yaml'
                     ),
            # Additional critical test cases
            TestData(pipeline_display_name='Artifact Cache Pipeline',
                     pipeline_name='artifact_cache-pipeline',
                     pipeline_func=artifact_cache_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='artifact_cache.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/artifact_cache.yaml'
                     ),
            TestData(pipeline_display_name='Artifact Crust Pipeline',
                     pipeline_name='artifact_crust-pipeline',
                     pipeline_func=artifact_crust_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='artifact_crust.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/artifact_crust.yaml'
                     ),
            TestData(pipeline_display_name='Container Component with No Inputs',
                     pipeline_name='v2-container-component-no-input',
                     pipeline_func=container_no_inputs_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='container_component_with_no_inputs.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/container_component_with_no_inputs.yaml'
                     ),
            TestData(pipeline_display_name='Loop Consume Upstream',
                     pipeline_name='loop-consume-upstream',
                     pipeline_func=loop_consume_upstream,
                     pipline_func_args=None,
                     compiled_file_name='loop_consume_upstream.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/loop_consume_upstream.yaml'
                     ),
            TestData(pipeline_display_name='Parameters Simple Pipeline',
                     pipeline_name='math-pipeline',
                     pipeline_func=parameters_simple_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='parameters_simple.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/parameters_simple.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Artifact Upload Download',
                     pipeline_name='pipeline-with-datasets',
                     pipeline_func=artifact_upload_download_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_artifact_upload_download.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_artifact_upload_download.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Input Status State',
                     pipeline_name='status-state-pipeline',
                     pipeline_func=status_state_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_input_status_state.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_input_status_state.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Placeholders',
                     pipeline_name='pipeline-with-placeholders',
                     pipeline_func=pipeline_with_placeholders,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_placeholders.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_placeholders.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Pod Metadata',
                     pipeline_name='pipeline-with-pod-metadata',
                     pipeline_func=pipeline_with_pod_metadata,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_pod_metadata.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_pod_metadata.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Secret as Env',
                     pipeline_name='pipeline-secret-env',
                     pipeline_func=pipeline_secret_env,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_secret_as_env.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_secret_as_env.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Workspace',
                     pipeline_name='pipeline-with-workspace',
                     pipeline_func=pipeline_with_workspace,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_workspace.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pipeline_with_workspace.yaml'
                     ),
            TestData(pipeline_display_name='Two Step Pipeline Containerized',
                     pipeline_name='containerized-two-step-pipeline',
                     pipeline_func=two_step_containerized_pipeline,
                     pipline_func_args={'text': 'Hello KFP Containerized!'},
                     compiled_file_name='two_step_pipeline_containerized.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/two_step_pipeline_containerized.yaml'
                     ),
            TestData(pipeline_display_name='Nested Pipeline Opt Input Child Level',
                     pipeline_name='nested-pipeline-opt-input-child-level',
                     pipeline_func=nested_pipeline_opt_input_child_level,
                     pipline_func_args=None,
                     compiled_file_name='nested_pipeline_opt_input_child_level.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/nested_pipeline_opt_input_child_level_compiled.yaml'
                     ),
            TestData(pipeline_display_name='Pythonic Artifacts Test Pipeline',
                     pipeline_name='split-datasets-and-return-first',
                     pipeline_func=pythonic_artifacts_test_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pythonic_artifacts_test_pipeline.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/pythonic_artifacts_test_pipeline.yaml'
                     ),
            # Additional valid directory test cases
            TestData(pipeline_display_name='Components with Optional Artifacts',
                     pipeline_name='optional-artifact-pipeline',
                     pipeline_func=optional_artifacts_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='components_with_optional_artifacts.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/components_with_optional_artifacts.yaml'
                     ),
            TestData(pipeline_display_name='Lightweight Python Functions Pipeline',
                     pipeline_name='my-test-pipeline-beta',
                     pipeline_func=lightweight_python_pipeline,
                     pipline_func_args={'message': 'Hello KFP!', 'input_dict': {'A': 1, 'B': 2}},
                     compiled_file_name='lightweight_python_functions_pipeline.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/lightweight_python_functions_pipeline.yaml'
                     ),
            TestData(pipeline_display_name='XGBoost Sample Pipeline',
                     pipeline_name='xgboost-sample-pipeline',
                     pipeline_func=xgboost_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='xgboost_sample_pipeline.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/xgboost_sample_pipeline.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with After Dependencies',
                     pipeline_name='pipeline-with-after',
                     pipeline_func=after_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_after.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_after.yaml'
                     ),
            TestData(pipeline_display_name='Metrics Visualization Pipeline',
                     pipeline_name='metrics-visualization-pipeline',
                     pipeline_func=metrics_visualization_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='metrics_visualization_v2.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/metrics_visualization_v2.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Nested Conditions',
                     pipeline_name='nested-conditions-pipeline',
                     pipeline_func=nested_conditions_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_nested_conditions.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_nested_conditions.yaml'
                     ),
            TestData(pipeline_display_name='Container IO Component',
                     pipeline_name='container-io',
                     pipeline_func=container_io,
                     pipline_func_args={'text': 'Hello Container!'},
                     compiled_file_name='container_io.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/container_io.yaml'
                     ),
            # Additional important pipeline test cases
            TestData(pipeline_display_name='Pipeline with Exit Handler',
                     pipeline_name='pipeline-with-exit-handler',
                     pipeline_func=exit_handler_pipeline,
                     pipline_func_args={'message': 'Hello Exit Handler!'},
                     compiled_file_name='pipeline_with_exit_handler.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_exit_handler.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Importer',
                     pipeline_name='pipeline-with-importer',
                     pipeline_func=importer_pipeline,
                     pipline_func_args={'dataset2': 'gs://ml-pipeline-playground/shakespeare2.txt'},
                     compiled_file_name='pipeline_with_importer.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_importer.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Nested Loops',
                     pipeline_name='pipeline-with-nested-loops',
                     pipeline_func=nested_loops_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_nested_loops.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_nested_loops.yaml'
                     ),
            TestData(pipeline_display_name='Concat Message Component',
                     pipeline_name='concat-message',
                     pipeline_func=concat_message,
                     pipline_func_args={'message1': 'Hello', 'message2': ' World!'},
                     compiled_file_name='concat_message.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/concat_message.yaml'
                     ),
            TestData(pipeline_display_name='Preprocess Component',
                     pipeline_name='preprocess',
                     pipeline_func=preprocess,
                     pipline_func_args={'message': 'test', 'input_dict_parameter': {'A': 1}, 'input_list_parameter': ['a', 'b']},
                     compiled_file_name='preprocess.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/preprocess.yaml'
                     ),
            TestData(pipeline_display_name='Sequential Pipeline V2',
                     pipeline_name='sequential',
                     pipeline_func=sequential,
                     pipline_func_args={'url': 'gs://sample-data/test.txt'},
                     compiled_file_name='sequential_v2.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/sequential-v2.yaml'
                     ),
            TestData(pipeline_display_name='Artifacts Simple Pipeline',
                     pipeline_name='math-pipeline',
                     pipeline_func=artifacts_simple_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='artifacts_simple.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/artifacts_simple.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Multiple Exit Handlers',
                     pipeline_name='pipeline-with-multiple-exit-handlers',
                     pipeline_func=multiple_exit_handlers_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_multiple_exit_handlers.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_multiple_exit_handlers.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Reused Component',
                     pipeline_name='pipeline-with-reused-component',
                     pipeline_func=reused_component_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_reused_component.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_reused_component.yaml'
                     ),
            # Additional missing pipeline test cases
            TestData(pipeline_display_name='Artifacts Complex Pipeline',
                     pipeline_name='math-pipeline',
                     pipeline_func=artifacts_complex_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='artifacts_complex.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/artifacts_complex.yaml'
                     ),
            TestData(pipeline_display_name='Conditional Producer and Consumers',
                     pipeline_name='math-pipeline',
                     pipeline_func=conditional_producer_consumers_pipeline,
                     pipline_func_args={'threshold': 2},
                     compiled_file_name='conditional_producer_and_consumers.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/conditional_producer_and_consumers.yaml'
                     ),
            TestData(pipeline_display_name='Collected Artifacts Pipeline',
                     pipeline_name='collected-artifact-pipeline',
                     pipeline_func=collected_artifact_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='collected_artifacts.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/collected_artifacts.yaml'
                     ),
            TestData(pipeline_display_name='Pythonic Artifacts with Multiple Returns',
                     pipeline_name='split-datasets-and-return-first',
                     pipeline_func=pythonic_artifacts_multiple_returns,
                     pipline_func_args=None,
                     compiled_file_name='pythonic_artifacts_with_multiple_returns.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pythonic_artifacts_with_multiple_returns.yaml'
                     ),
            TestData(pipeline_display_name='Identity Component',
                     pipeline_name='identity',
                     pipeline_func=identity,
                     pipline_func_args={'value': 'test'},
                     compiled_file_name='identity.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/identity.yaml'
                     ),
            TestData(pipeline_display_name='Input Artifact Component',
                     pipeline_name='input-artifact',
                     pipeline_func=input_artifact,
                     pipline_func_args=None,
                     compiled_file_name='input_artifact.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/input_artifact.yaml'
                     ),
            # Final batch of remaining missing pipeline test cases
            TestData(pipeline_display_name='Nested Return Component',
                     pipeline_name='nested-return',
                     pipeline_func=nested_return,
                     pipline_func_args=None,
                     compiled_file_name='nested_return.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/nested_return.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline in Pipeline',
                     pipeline_name='pipeline-in-pipeline',
                     pipeline_func=pipeline_in_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_in_pipeline.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_in_pipeline.yaml'
                     ),
            TestData(pipeline_display_name='Container with Concat Placeholder',
                     pipeline_name='container-with-concat-placeholder',
                     pipeline_func=container_with_concat_placeholder,
                     pipline_func_args={'text1': 'Hello'},
                     compiled_file_name='container_with_concat_placeholder.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/container_with_concat_placeholder.yaml'
                     ),
            TestData(pipeline_display_name='Lightweight Python Functions with Outputs',
                     pipeline_name='my-test-pipeline-output',
                     pipeline_func=lightweight_python_with_outputs_pipeline,
                     pipline_func_args={'first_message': 'Hello KFP!', 'second_message':'Welcome', 'first_number': 3, 'second_number': 4},
                     compiled_file_name='lightweight_python_functions_with_outputs.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/lightweight_python_functions_with_outputs.yaml'
                     ),
            TestData(pipeline_display_name='Output Metrics Component',
                     pipeline_name='output-metrics',
                     pipeline_func=output_metrics,
                     pipline_func_args=None,
                     compiled_file_name='output_metrics.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/output_metrics.yaml'
                     ),
            TestData(pipeline_display_name='Dict Input Component',
                     pipeline_name='dict-input',
                     pipeline_func=dict_input,
                     pipline_func_args={'struct': {'key1': 'value1', 'key2': 'value2'}},
                     compiled_file_name='dict_input.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/dict_input.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Volume',
                     pipeline_name='pipeline-with-volume',
                     pipeline_func=pipeline_with_volume,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_volume.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_volume.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Volume No Cache',
                     pipeline_name='pipeline-with-volume-no-cache',
                     pipeline_func=pipeline_with_volume_no_cache,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_volume_no_cache.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_volume_no_cache.yaml'
                     ),
            TestData(pipeline_display_name='Container with If Placeholder',
                     pipeline_name='container-with-if-placeholder',
                     pipeline_func=container_with_if_placeholder,
                     pipline_func_args=None,
                     compiled_file_name='container_with_if_placeholder.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/container_with_if_placeholder.yaml'
                     ),
            TestData(pipeline_display_name='Container with Placeholder in F-String',
                     pipeline_name='container-with-placeholder-in-fstring',
                     pipeline_func=container_with_placeholder_in_fstring,
                     pipline_func_args=None,
                     compiled_file_name='container_with_placeholder_in_fstring.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/container_with_placeholder_in_fstring.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline in Pipeline Complex',
                     pipeline_name='pipeline-in-pipeline-complex',
                     pipeline_func=pipeline_in_pipeline_complex,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_in_pipeline_complex.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_in_pipeline_complex.yaml'
                     ),
            # Additional high-priority missing pipeline test cases
            TestData(pipeline_display_name='Lucky Number Pipeline (If Elif Else Complex)',
                     pipeline_name='lucky-number-pipeline',
                     pipeline_func=lucky_number_pipeline,
                     pipline_func_args={'add_drumroll': True, 'repeat_if_lucky_number': True, 'trials': [1, 2, 3]},
                     compiled_file_name='if_elif_else_complex.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/if_elif_else_complex.yaml'
                     ),
            TestData(pipeline_display_name='If Elif Else with OneOf Parameters',
                     pipeline_name='if-elif-else-with-oneof-parameters',
                     pipeline_func=if_elif_else_oneof_params_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='if_elif_else_with_oneof_parameters.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/if_elif_else_with_oneof_parameters.yaml'
                     ),
            TestData(pipeline_display_name='If Else with OneOf Parameters',
                     pipeline_name='if-else-with-oneof-parameters',
                     pipeline_func=if_else_oneof_params_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='if_else_with_oneof_parameters.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/if_else_with_oneof_parameters.yaml'
                     ),
            TestData(pipeline_display_name='If Else with OneOf Artifacts',
                     pipeline_name='if-else-with-oneof-artifacts',
                     pipeline_func=if_else_oneof_artifacts_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='if_else_with_oneof_artifacts.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/if_else_with_oneof_artifacts.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Loops and Conditions',
                     pipeline_name='pipeline-with-loops-and-conditions',
                     pipeline_func=loops_and_conditions_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_loops_and_conditions.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_loops_and_conditions.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Various IO Types',
                     pipeline_name='pipeline-with-various-io-types',
                     pipeline_func=various_io_types_pipeline,
                     pipline_func_args={'input1': 'Hello', 'input4': 'World'},
                     compiled_file_name='pipeline_with_various_io_types.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_various_io_types.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Metrics Outputs',
                     pipeline_name='pipeline-with-metrics-outputs',
                     pipeline_func=metrics_outputs_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_metrics_outputs.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_metrics_outputs.yaml'
                     ),
            TestData(pipeline_display_name='Containerized Python Component',
                     pipeline_name='containerized-concat-message',
                     pipeline_func=containerized_concat_message,
                     pipline_func_args={'message1': 'Hello', 'message2': ' Containerized!'},
                     compiled_file_name='containerized_python_component.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/containerized_python_component.yaml'
                     ),
            TestData(pipeline_display_name='Container with Artifact Output',
                     pipeline_name='container-with-artifact-output',
                     pipeline_func=container_with_artifact_output,
                     pipline_func_args={'num_epochs': 10},
                     compiled_file_name='container_with_artifact_output.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/container_with_artifact_output.yaml'
                     ),
            TestData(pipeline_display_name='Nested with Parameters Pipeline',
                     pipeline_name='math-pipeline',
                     pipeline_func=nested_with_parameters_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='nested_with_parameters.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/nested_with_parameters.yaml'
                     ),
            TestData(pipeline_display_name='Parameters Complex Pipeline',
                     pipeline_name='math-pipeline',
                     pipeline_func=parameters_complex_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='parameters_complex.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/parameters_complex.yaml'
                     ),
            # Additional critical missing pipeline test cases
            TestData(pipeline_display_name='Component with Metadata Fields',
                     pipeline_name='dataset-joiner',
                     pipeline_func=dataset_joiner,
                     pipline_func_args=None,
                     compiled_file_name='component_with_metadata_fields.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/component_with_metadata_fields.yaml'
                     ),
            TestData(pipeline_display_name='Component with Pip Index URLs',
                     pipeline_name='my-test-pipeline',
                     pipeline_func=pip_index_urls_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='component_with_pip_index_urls.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/component_with_pip_index_urls.yaml'
                     ),
            TestData(pipeline_display_name='Flip Coin Pipeline',
                     pipeline_name='flip-coin',
                     pipeline_func=flip_coin,
                     pipline_func_args=None,
                     compiled_file_name='flip_coin.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/flip_coin.yaml'
                     ),
            TestData(pipeline_display_name='Component with Pip Install in Venv',
                     pipeline_name='component-with-pip-install-in-venv',
                     pipeline_func=pip_install_venv_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='component_with_pip_install_in_venv.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/component_with_pip_install_in_venv.yaml'
                     ),
            TestData(pipeline_display_name='Component with Pip Install',
                     pipeline_name='component-with-pip-install',
                     pipeline_func=pip_install_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='component_with_pip_install.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/component_with_pip_install.yaml'
                     ),
            TestData(pipeline_display_name='Component with Task Final Status',
                     pipeline_name='component-with-task-final-status',
                     pipeline_func=task_final_status_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='component_with_task_final_status_GH-12033.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/component_with_task_final_status_GH-12033.yaml'
                     ),
            TestData(pipeline_display_name='Producer Consumer Parallel For Pipeline',
                     pipeline_name='math-pipeline',
                     pipeline_func=producer_consumer_parallel_for_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_producer_consumer.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_producer_consumer.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Secret as Volume',
                     pipeline_name='pipeline-with-secret-as-volume',
                     pipeline_func=pipeline_secret_volume,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_secret_as_volume.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_secret_as_volume.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Utils',
                     pipeline_name='pipeline-with-utils',
                     pipeline_func=pipeline_with_utils,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_utils.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_utils.yaml'
                     ),
            # Additional missing pipeline test cases
            TestData(pipeline_display_name='Arguments Parameters Pipeline',
                     pipeline_name='echo',
                     pipeline_func=arguments_parameters_echo,
                     pipline_func_args={'param1': 'hello', 'param2': 'world'},
                     compiled_file_name='arguments_parameters.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/arguments-parameters.yaml'
                     ),
            TestData(pipeline_display_name='Container No Input',
                     pipeline_name='container-no-input',
                     pipeline_func=container_no_input,
                     pipline_func_args=None,
                     compiled_file_name='container_no_input.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/container_no_input.yaml'
                     ),
            TestData(pipeline_display_name='Environment Variable Test',
                     pipeline_name='test-env-exists',
                     pipeline_func=test_env_exists,
                     pipline_func_args={'env_var': 'HOME'},
                     compiled_file_name='env_var.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/env-var.yaml'
                     ),
            TestData(pipeline_display_name='Create Pod Metadata Complex',
                     pipeline_name='create-pod-metadata-complex',
                     pipeline_func=create_pod_metadata_complex,
                     pipline_func_args=None,
                     compiled_file_name='create_pod_metadata_complex.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/create_pod_metadata_complex.yaml'
                     ),
            TestData(pipeline_display_name='Nested Pipeline Opt Inputs Nil',
                     pipeline_name='nested-pipeline-opt-inputs-nil',
                     pipeline_func=nested_pipeline_opt_inputs_nil,
                     pipline_func_args=None,
                     compiled_file_name='nested_pipeline_opt_inputs_nil.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/nested_pipeline_opt_inputs_nil_compiled.yaml'
                     ),
            TestData(pipeline_display_name='Nested Pipeline Opt Inputs Parent Level',
                     pipeline_name='nested-pipeline-opt-inputs-parent-level',
                     pipeline_func=nested_pipeline_opt_inputs_parent_level,
                     pipline_func_args=None,
                     compiled_file_name='nested_pipeline_opt_inputs_parent_level.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/critical/nested_pipeline_opt_inputs_parent_level_compiled.yaml'
                     ),
            # Final remaining missing pipeline test cases
            TestData(pipeline_display_name='Cross Loop After Topology',
                     pipeline_name='cross-loop-after-topology',
                     pipeline_func=cross_loop_after_topology_pipeline,
                     pipline_func_args=None,
                     compiled_file_name='cross_loop_after_topology.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/cross_loop_after_topology.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline as Exit Task',
                     pipeline_name='pipeline-with-task-final-status-conditional',
                     pipeline_func=pipeline_as_exit_task,
                     pipline_func_args={'message': 'Hello Exit Task!'},
                     compiled_file_name='pipeline_as_exit_task.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_as_exit_task.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline in Pipeline Loaded from YAML',
                     pipeline_name='pipeline-in-pipeline-loaded-from-yaml',
                     pipeline_func=pipeline_in_pipeline_loaded_from_yaml,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_in_pipeline_loaded_from_yaml.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_in_pipeline_loaded_from_yaml.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Dynamic Importer Metadata',
                     pipeline_name='dynamic-importer-metadata-pipeline',
                     pipeline_func=pipeline_with_dynamic_importer_metadata,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_dynamic_importer_metadata.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_dynamic_importer_metadata.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Google Artifact Type',
                     pipeline_name='pipeline-with-google-artifact-types',
                     pipeline_func=pipeline_with_google_artifact_type,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_google_artifact_type.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_google_artifact_type.yaml'
                     ),
            TestData(pipeline_display_name='Pipeline with Metadata Fields',
                     pipeline_name='pipeline-with-metadata-fields',
                     pipeline_func=pipeline_with_metadata_fields,
                     pipline_func_args=None,
                     compiled_file_name='pipeline_with_metadata_fields.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pipeline_with_metadata_fields.yaml'
                     ),
            TestData(pipeline_display_name='Pythonic Artifacts with List of Artifacts',
                     pipeline_name='make-and-join-datasets',
                     pipeline_func=pythonic_artifacts_with_list_of_artifacts,
                     pipline_func_args={'texts': ['text1', 'text2']},
                     compiled_file_name='pythonic_artifacts_with_list_of_artifacts.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pythonic_artifacts_with_list_of_artifacts.yaml'
                     ),
            TestData(pipeline_display_name='Pythonic Artifact with Single Return',
                     pipeline_name='make-language-model-pipeline',
                     pipeline_func=pythonic_artifact_with_single_return,
                     pipline_func_args=None,
                     compiled_file_name='pythonic_artifact_with_single_return.yaml',
                     expected_compiled_file_path=f'{_VALID_PIPELINE_FILES}/pythonic_artifact_with_single_return.yaml'
                     ),
        ],
        ids=str)
    def test_compilation(self, pipeline_data: TestData):
        temp_compiled_pipeline_file = os.path.join(tempfile.gettempdir(), pipeline_data.compiled_file_name)

        compiler.Compiler().compile(
            pipeline_func=pipeline_data.pipeline_func,
            pipeline_name=pipeline_data.pipeline_name,
            pipeline_parameters=pipeline_data.pipline_func_args,
            package_path=temp_compiled_pipeline_file,
        )
        print(f'Pipeline Created at : {temp_compiled_pipeline_file}')
        print(f'Parsing expected yaml {pipeline_data.expected_compiled_file_path} for comparison')
        expected_pipeline_specs, expected_platform_specs = FileUtils.read_yaml_file(pipeline_data.expected_compiled_file_path)
        print(f'Parsing compiled yaml {temp_compiled_pipeline_file} for comparison')
        generated_pipeline_specs, generated_platform_specs = FileUtils.read_yaml_file(temp_compiled_pipeline_file)
        print('Verify that the generated yaml matches expected yaml or not')
        ComparisonUtils.compare_pipeline_spec_dicts(
            actual=generated_pipeline_specs,
            expected=expected_pipeline_specs,
            display_name=pipeline_data.pipeline_display_name,
            name=pipeline_data.pipeline_name,
            runtime_params=pipeline_data.pipline_func_args,
        )
        ComparisonUtils.compare_pipeline_spec_dicts(
            actual=generated_platform_specs,
            expected=expected_platform_specs)