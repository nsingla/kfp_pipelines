import asyncio
import datetime

import time

from kfp_server_api import V2beta1RecurringRun, V2beta1RuntimeState
from config.test_config import TestConfig
from models.test_scenario import TestScenario

from runners.base_runner import BaseRunner

class PipelineScheduledRunner(BaseRunner):

    experiment_id: str = None
    polling_wait_time_for_run_to_finish: int = 5000

    def __init__(self, test_scenario: TestScenario):
        super().__init__(test_scenario)
        self.runs_created: list[str] = list()

    def create_recurring_run(self) -> V2beta1RecurringRun:
        pipeline_file_path = f"{TestConfig.pipeline_files_directory}/{self.test_scenario.pipeline_file}"
        job_name = f"PerfTestScheduledRun-{time.time()}"
        self.logger.info(f"Creating Scheduled Pipeline Run under experiment ID: {self.experiment_id}")
        run = self.call_and_capture_time("CreateRecurringRun", self.kfp_client.create_recurring_run, job_name=job_name, pipeline_package_path=pipeline_file_path, experiment_id=self.experiment_id, enable_caching=TestConfig.CACHE_ENABLED, params=self.test_scenario.params, cron_expression=self.test_scenario.cron)
        self.logger.info(f"Run {run.run_id} created for Pipeline under experiment ID: {self.experiment_id}")
        self.runs_created.append(run.run_id)
        return run

    async def wait_for_run_to_finish(self, run_id: str):
        pipeline_state = self.kfp_client.get_run(run_id).state
        while pipeline_state in [V2beta1RuntimeState.RUNNING, V2beta1RuntimeState.CANCELING, V2beta1RuntimeState.PENDING]:
            self.logger.info(f"Waiting for Pipeline Run with ID: {run_id}, to finish, currently in state: {pipeline_state}... Waiting for {self.polling_wait_time_for_run_to_finish}ms")
            await asyncio.sleep(self.polling_wait_time_for_run_to_finish)
            pipeline_state = self.kfp_client.get_run(run_id).state

    async def upload_and_run_pipeline(self):
        created_run = self.create_recurring_run()
        await self.call_and_capture_time("RecurringRunCompletionTime", self.wait_for_run_to_finish, run_id=created_run.run_id)

    async def stop(self):
        while datetime.datetime.now() < self.test_end_date:
            runs_not_completed = [run for run in self.kfp_client.list_runs(page_size=10000).runs if run.state in [V2beta1RuntimeState.RUNNING, V2beta1RuntimeState.CANCELING, V2beta1RuntimeState.PENDING] and run.run_id in [self.runs_created]]
            runs_to_run = self.test_scenario.num_times - len(runs_not_completed)
            async_tasks = []
            for run_number in range(runs_to_run):
                task = asyncio.create_task(self.upload_and_run_pipeline())
                async_tasks.append(task)
            await asyncio.gather(async_tasks)
        self.metricsToReturn[self.metric_run_times] = self.elapsed_times

    def run(self):
        self.start()
        asyncio.run(self.stop())
        return self.metricsToReturn
