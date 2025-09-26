import os
import yaml


class FileUtils:

    PROJECT_ROOT = os.path.abspath(os.path.join(__file__, *([os.path.pardir] * 5)))
    TEST_DATA = os.path.join(PROJECT_ROOT, "test_data")
    VALID_PIPELINE_FILES = os.path.join(TEST_DATA, "pipeline_files", "valid")
    COMPONENTS = os.path.join(TEST_DATA, "components")

    @classmethod
    def read_yaml_file(cls, filepath) -> tuple:
        """
        Read the pipeline spec file at the specific file path and parse it into a dict and return a tuple of (pipeline_spec, platform_spec)
        :param filepath:
        :return:
        """
        pipeline_specs: dict = None
        platform_specs: dict = None
        with open(filepath, 'r') as file:
            try:
                yaml_data = yaml.safe_load_all(file)
                for data in yaml_data:
                    if 'pipelineInfo' in data.keys():
                        pipeline_specs = data
                    else:
                        platform_specs = data
                return pipeline_specs, platform_specs
            except yaml.YAMLError as ex:
                print(f'Error parsing YAML file: {ex}')
                raise f'Could not load yaml file: {filepath} due to {ex}'

