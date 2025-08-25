import datetime
import random

import time

from kfp_server_api import V2beta1Pipeline, V2beta1PipelineVersion, V2beta1Run, V2beta1Experiment
from config.test_config import TestConfig
from models.test_scenario import TestScenario

from base_runner import BaseRunner

class RandomGetRunner(BaseRunner):
    page_size: int = 1000

    def __init__(self, test_scenario: TestScenario):
        super().__init__(test_scenario)
        self.metricsToReturn: dict[str, list] = dict()

    def call_and_capture_time(self, metric_name: str, func, *args, **kwargs):
        start = time.perf_counter()
        result = func(*args, **kwargs)
        elapsed_time = time.perf_counter() - start
        self.metricsToReturn[metric_name].append(elapsed_time)
        return result

    async def get_random_pipeline(self) -> V2beta1Pipeline:
        self.logger.info(f"Listing Pipelines")
        pipelines: list[V2beta1Pipeline] = self.call_and_capture_time("ListPipelines", self.kfp_client.list_pipelines, page_size=self.page_size, namespace=TestConfig.NAMESPACE).pipelines
        self.logger.info(f"Selecting random Pipeline from the list")
        return random.choice(pipelines)

    async def list_pipeline_versions(self, pipeline_id: str) -> list[V2beta1PipelineVersion]:
        self.logger.info(f"Listing Pipeline Version")
        return self.call_and_capture_time("ListPipelineVersions", self.kfp_client.list_pipeline_versions, page_size=self.page_size, pipeline_id=pipeline_id).pipeline_versions

    async def get_random_experiment(self) -> V2beta1Experiment:
        self.logger.info(f"Listing Experiments")
        runs: list[V2beta1Experiment] = self.call_and_capture_time("ListExperiment", self.kfp_client.list_experiments, page_size=self.page_size, namespace=TestConfig.NAMESPACE).experiments
        self.logger.info(f"Selecting random Experiment from the list")
        return random.choice(runs)

    async def get_run(self, experiment_id: str) -> V2beta1Run:
        self.logger.info(f"Listing Runs")
        runs: list[V2beta1Run] = self.call_and_capture_time("ListRuns", self.kfp_client.list_runs, page_size=self.page_size, experiment_id=experiment_id, namespace=TestConfig.NAMESPACE).runs
        self.logger.info(f"Selecting random Pipeline Run from the list")
        return random.choice(runs)

    async def get_run_details(self, run_id: str) -> V2beta1Run:
        self.logger.info(f"Get run details for run_id={run_id}")
        return self.call_and_capture_time("GetRun", self.kfp_client.get_run, run_id=run_id)

    async def stop(self):
        while datetime.datetime.now() < self.test_end_date:
            pipeline = await self.get_random_pipeline()
            await self.list_pipeline_versions(pipeline.pipeline_id)
            experiment = await self.get_random_experiment()
            run = await self.get_run(experiment_id=experiment.experiment_id)
            await self.get_run_details(run.run_id)

    def run(self):
        self.start()
        self.stop()
        return self.metricsToReturn
