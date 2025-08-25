import asyncio
import datetime
import random

import time

from kfp_server_api import V2beta1Pipeline, V2beta1PipelineVersion, V2beta1Run, V2beta1RuntimeState, V2beta1Experiment
from config.test_config import TestConfig
from models.test_scenario import TestScenario

from enums.test_mode import TestMode
from base_runner import BaseRunner

class PipelineRunner(BaseRunner):

    polling_wait_time_for_run_to_finish: int = 5000
    metric_run_times: str = "RunTimes"

    def __init__(self, test_scenario: TestScenario):
        super().__init__(test_scenario)
        if self.test_scenario.mode == TestMode.EXPERIMENT:
            experiment_name: str = f"PerfTestExperiment-{str(time.time())}"
            self.experiment_id = self.kfp_client.create_experiment(name=experiment_name, description="Experiment to capture performance test pipeline runs", namespace=TestConfig.NAMESPACE).experiment_id
        else:
            experiments: list[V2beta1Experiment] = self.kfp_client.list_experiments(page_size=1000).experiments
            for exp in experiments:
                if exp.display_name.lower() == "default":
                    self.experiment_id = exp.experiment_id
        self.metricsToReturn: dict[str, list] = dict()
        self.elapsed_times: list[float] = list()
        self.runs_created: list[str] = list()

    async def upload_pipeline(self) -> V2beta1Pipeline:
        pipeline_name: str = f"PerfTestPipeline-{str(time.time())}"
        self.logger.info(f"Uploading Pipeline with name: {pipeline_name}")
        pipeline = self.kfp_client.upload_pipeline(pipeline_name=pipeline_name, pipeline_package_path=f"{TestConfig.test_data_dir}/{self.test_scenario}", namespace=TestConfig.NAMESPACE)
        self.logger.info(f"Uploaded Pipeline ID: {pipeline.pipeline_id}")
        return pipeline

    async def run_pipeline(self, pipeline_id: str) -> V2beta1Run:
        pipeline_versions:  list[V2beta1PipelineVersion] = self.kfp_client.list_pipeline_versions(pipeline_id=pipeline_id).pipeline_versions
        random_pipeline_version: V2beta1PipelineVersion = random.choice(pipeline_versions)
        self.logger.info(f"Creating Pipeline Run for Pipeline with ID: {pipeline_id}, with version ID: {random_pipeline_version} and experiment ID: {self.experiment_id}")
        run = self.kfp_client.run_pipeline(pipeline_id=pipeline_id, version_id=random_pipeline_version.pipeline_version_id, experiment_id=self.experiment_id, enable_caching=TestConfig.CACHE_ENABLED, params=self.test_scenario.params)
        self.logger.info(f"Run {run.run_id} created for Pipeline with ID: {pipeline_id}, with version ID: {random_pipeline_version} and experiment ID: {self.experiment_id}")
        self.runs_created.append(run.run_id)
        return run

    async def wait_for_run_to_finish(self, run_id: str):
        pipeline_state = self.kfp_client.get_run(run_id).state
        while pipeline_state in [V2beta1RuntimeState.RUNNING, V2beta1RuntimeState.CANCELING, V2beta1RuntimeState.PENDING]:
            self.logger.info(f"Waiting for Pipeline Run with ID: {run_id}, to finish, currently in state: {pipeline_state}... Waiting for {self.polling_wait_time_for_run_to_finish}ms")
            await asyncio.sleep(self.polling_wait_time_for_run_to_finish)
            pipeline_state = self.kfp_client.get_run(run_id).state

    async def upload_and_run_pipeline(self, num_pipelines):
        for pipeline_counter in range(num_pipelines):
            created_pipeline = self.upload_pipeline()
            created_run = self.run_pipeline(created_pipeline.pipeline_id)
            start = time.perf_counter()
            await asyncio.gather(self.wait_for_run_to_finish(created_run.run_id))
            elapsed = time.perf_counter() - start
            self.elapsed_times.append(elapsed)
            yield pipeline_counter

    async def run_pipelines(self, num_of_runs):
        runs_not_completed = [run for run in self.kfp_client.list_runs(page_size=10000).runs if run.state in [V2beta1RuntimeState.RUNNING, V2beta1RuntimeState.CANCELING, V2beta1RuntimeState.PENDING] and run.run_id in [self.runs_created]]
        runs_to_run = num_of_runs - len(runs_not_completed)
        async for item in self.upload_and_run_pipeline(runs_to_run):
            self.logger.info(f"Running pipeline number: {item}")

    async def stop(self):
        while datetime.datetime.now() < self.test_end_date:
            await self.run_pipelines(self.test_scenario.num_times)
        self.metricsToReturn[self.metric_run_times] = self.elapsed_times

    def run(self):
        self.start()
        self.stop()
        return self.metricsToReturn
