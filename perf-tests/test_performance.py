from concurrent.futures import ThreadPoolExecutor, as_completed, Future
from datetime import datetime
from runners import base_runner, pipeline_runner
from models.test_scenario import TestScenario
from utils.json_utils import JsonDeserializationUtils
from enums.test_mode import TestMode
from logger import logger
from logging import Logger

import pytest
from config.test_config import TestConfig

logger: Logger = logger.Logger(True).logger


class TestPerformance:
    """
    pytest class contains the test method to run multithreading script.
    """


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
            base_runner.BaseRunner.test_start = datetime.now()

            with (ThreadPoolExecutor(max_workers=len(scenarios)) as executor):
                futures = []
                thread_number = 0
                for scenario in scenarios:
                    thread_number += 1
                    logger.info("Running Scenario: " + scenario.mode.name)
                    if scenario.mode == TestMode.PIPELINE_RUN or scenario.mode == TestMode.EXPERIMENT:
                        logger.info(
                            f"Thread {thread_number}: Run Pipeline Operation: {scenario.model_dump(exclude_none=True)}")
                        pipeline_uploader_runner = pipeline_runner.PipelineRunner(scenario)
                        futures.append(executor.submit(pipeline_uploader_runner.run))
            for future in as_completed(futures):
                logger.info(future.result())
        except Exception as e:
            test_failed = True
            exception_to_throw = e

        if test_failed:
            raise exception_to_throw