from concurrent.futures import ThreadPoolExecutor, as_completed, Future
from datetime import datetime
from runners.base_runner import BaseRunner
from runners.pipeline_runner import PipelineRunner
from runners.random_get_runner import RandomGetRunner
from models.test_scenario import TestScenario
from utils.json_utils import JsonDeserializationUtils
from enums.test_mode import TestMode
from logger import logger
from logging import Logger

import pytest
from config.test_config import TestConfig
from factory.client_factory import ClientFactory

logger = logger.Logger(True)
colored_logger: Logger = logger.logger


class TestPerformance:
    """
    pytest class contains the test method to run multithreading script.
    """

    @pytest.fixture(autouse=True)
    def setup_and_teardown(self):
        """
        Setup method to run before each test.
        """
        BaseRunner.test_start = datetime.now()
        colored_logger.info("Setting up test environment")
        yield
        colored_logger.info(f"Test completed in {datetime.now() - BaseRunner.test_start}")
        logger.shutdown_handler()

    def test_nelesh(self):
        ClientFactory().kfp_client.create_run_from_pipeline_package(f"{TestConfig.pipeline_files_directory}/add_numbers.yaml", enable_caching=TestConfig.CACHE_ENABLED, arguments={"a":4,"b":5})

    @pytest.mark.performance
    def test_scenario(self):
        """
        Test method to run all the runners in a multithreading environment. based on the input json provided.
        """
        test_failed = False
        exception_to_throw = None
        try:
            scenarios: list[TestScenario] = JsonDeserializationUtils.get_list_from_file(f"{TestConfig.test_scenario_directory}/{TestConfig.test_scenario_file_name}",
                                                                                          TestScenario)

            with (ThreadPoolExecutor(max_workers=len(scenarios)) as executor):
                futures = []
                thread_number = 0
                for scenario in scenarios:
                    thread_number += 1
                    colored_logger.info("Running Scenario: " + scenario.mode.name)
                    if scenario.mode == TestMode.PIPELINE_RUN or scenario.mode == TestMode.EXPERIMENT:
                        colored_logger.info(
                            f"Thread {thread_number}: Run Pipeline Operation: {scenario.model_dump(exclude_none=True)}")
                        pipeline_uploader_runner = PipelineRunner(scenario)
                        futures.append(executor.submit(pipeline_uploader_runner.run))
                    elif scenario.mode == TestMode.RANDOM_GETS:
                        colored_logger.info(
                            f"Thread {thread_number}: Run Pipeline Operation: {scenario.model_dump(exclude_none=True)}")
                        random_runner = RandomGetRunner(scenario)
                        futures.append(executor.submit(random_runner.run))
            for future in as_completed(futures):
                colored_logger.info(future.result())
        except Exception as e:
            test_failed = True
            exception_to_throw = e

        if test_failed:
            raise exception_to_throw