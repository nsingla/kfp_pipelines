import os
from pathlib import Path


class TestConfig:
    KFP_API_HOST: str = os.getenv("API_HOST", "localhost")
    KFP_API_PORT: str = os.getenv("API_PORT", "8888")
    IS_KUBEFLOW_MODE: bool = bool(os.getenv("isKubeflowMode", "False"))

    KFP_url = f"http://{KFP_API_HOST}:{KFP_API_PORT}"
    NAMESPACE: str = os.getenv("NAMESPACE", "kubeflow")
    LOCAL: bool = bool(os.getenv("local", "True"))
    CACHE_ENABLED: bool = bool(os.getenv("cacheEnabled", "True"))

    # Test Files
    current_directory = Path.cwd()
    test_data_dir = f"{current_directory}/test_data"
    test_scenario_directory = f"{test_data_dir}/scenarios"
    pipeline_files_directory = f"{test_data_dir}/pipeline_files"
    test_scenario_file_name: str = os.getenv("SCENARIO_FILE_NAME", "smoke.json")