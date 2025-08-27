import asyncio
import datetime

import time

from kfp_server_api import V2beta1Pipeline
from config.test_config import TestConfig
from models.test_scenario import TestScenario

from runners.base_runner import BaseRunner

class PipelineUploadRunner(BaseRunner):
    page_size: int = 1000

    def __init__(self, test_scenario: TestScenario):
        super().__init__(test_scenario)
        self.metricsToReturn: dict[str, list] = dict()


    def upload_pipeline(self) -> V2beta1Pipeline:
        self.logger.info(f"Uploading Pipeline")
        pipeline_file_path = f"{TestConfig.pipeline_files_directory}/{self.test_scenario.pipeline_file}"
        pipeline_name = f"PerfTestPipeline-{time.time()}"
        return self.call_and_capture_time("UploadPipeline", self.kfp_client.upload_pipeline, pipeline_package_path=pipeline_file_path, pipeline_name=pipeline_name, namespace=TestConfig.NAMESPACE).pipelines

    def stop(self):
        while datetime.datetime.now() < self.test_end_date:
            pipeline = self.upload_pipeline()
            self.logger.info(f"Uploaded Pipeline with id={pipeline.pipeline_id}")
            wait_time = (self.test_scenario.run_time/self.test_scenario.num_times) * 60
            time.sleep(wait_time)

    def run(self):
        self.start()
        self.stop()
        return self.metricsToReturn
