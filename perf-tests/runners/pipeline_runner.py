import asyncio
import datetime
import os.path
import random

import time

from kfp_server_api import V2beta1Pipeline, V2beta1PipelineVersion, V2beta1Run, V2beta1RuntimeState, V2beta1Experiment
from config.test_config import TestConfig
from models.test_scenario import TestScenario

from enums.test_mode import TestMode
from runners.base_runner import BaseRunner

class PipelineRunner(BaseRunner):

    experiment_id: str = None
    polling_wait_time_for_run_to_finish: int = 5000
    metric_run_times: str = "RunTimes"

    def __init__(self, test_scenario: TestScenario):
        super().__init__(test_scenario)
        self.metricsToReturn: dict[str, list] = dict()
        self.elapsed_times: list[float] = list()
        self.runs_created: list[str] = list()

    def run_pipeline(self) -> V2beta1Run:
        pipeline_file_path = f"{TestConfig.pipeline_files_directory}/{self.test_scenario.pipeline_file}"
        self.logger.info(f"Creating Pipeline Run under experiment ID: {self.experiment_id}")
        run = self.kfp_client.create_run_from_pipeline_package(pipeline_file_path, experiment_id=self.experiment_id, enable_caching=TestConfig.CACHE_ENABLED, arguments=self.test_scenario.params)
        self.logger.info(f"Run {run.run_id} created for Pipeline under experiment ID: {self.experiment_id}")
        self.runs_created.append(run.run_id)
        return run

    async def wait_for_run_to_finish(self, run_id: str):
        pipeline_state = self.kfp_client.get_run(run_id).state
        while pipeline_state in [V2beta1RuntimeState.RUNNING, V2beta1RuntimeState.CANCELING, V2beta1RuntimeState.PENDING]:
            self.logger.info(f"Waiting for Pipeline Run with ID: {run_id}, to finish, currently in state: {pipeline_state}... Waiting for {self.polling_wait_time_for_run_to_finish}ms")
            await asyncio.sleep(self.polling_wait_time_for_run_to_finish)
            pipeline_state = self.kfp_client.get_run(run_id).state

    async def upload_and_run_pipeline(self, num_pipelines):
        created_run = self.run_pipeline()
        start = time.perf_counter()
        await self.wait_for_run_to_finish(created_run.run_id)
        elapsed = time.perf_counter() - start
        self.elapsed_times.append(elapsed)

    async def stop(self):
        while datetime.datetime.now() < self.test_end_date:
            runs_not_completed = [run for run in self.kfp_client.list_runs(page_size=10000).runs if run.state in [V2beta1RuntimeState.RUNNING, V2beta1RuntimeState.CANCELING, V2beta1RuntimeState.PENDING] and run.run_id in [self.runs_created]]
            runs_to_run = self.test_scenario.num_times - len(runs_not_completed)
            async_tasks = []
            for run_number in range(runs_to_run):
                task = asyncio.create_task(self.upload_and_run_pipeline(run_number))
                async_tasks.append(task)
            await asyncio.gather(async_tasks)
        self.metricsToReturn[self.metric_run_times] = self.elapsed_times

    def run(self):
        self.start()
        if self.test_scenario.mode == TestMode.EXPERIMENT:
            experiment_name: str = f"PerfTestExperiment-{str(time.time())}"
            self.experiment_id = self.kfp_client.create_experiment(name=experiment_name, description="Experiment to capture performance test pipeline runs", namespace=TestConfig.NAMESPACE).experiment_id
        else:
            experiments: list[V2beta1Experiment] = self.kfp_client.list_experiments(page_size=1000).experiments
            for exp in experiments:
                if exp.display_name.lower() == "default":
                    self.experiment_id = exp.experiment_id
                    break
        asyncio.run(self.stop())
        return self.metricsToReturn
