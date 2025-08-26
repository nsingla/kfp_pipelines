import time
from abc import abstractmethod
import datetime

from config.test_config import TestConfig
from logging import Logger
from logger import logger
from models.test_scenario import TestScenario
from factory.client_factory import ClientFactory

logger: Logger = logger.Logger().logger


class BaseRunner:
    """
    This is a base runner class to get required common methods for all runners.
    """

    test_start = None

    def __init__(self, test_scenario: TestScenario):
        """
        Constructor for Base runner class which is the parent call for all the runners.
        :param test_scenario: the path of the scenario json file.
        """

        client_factory = ClientFactory()
        self.logger = client_factory.logger
        self.kfp_client = client_factory.kfp_client
        self.test_scenario = test_scenario
        self.test_start_date = self.test_start + datetime.timedelta(minutes=test_scenario.start_time)
        self.test_end_date = self.test_start_date + datetime.timedelta(minutes=test_scenario.run_time)

    def start(self):
        """
        This is the method to calculate the start time of the operation based on the value provided in input json.
        """
        self.logger.info(f"Start Date {self.test_start_date}")
        if self.test_start_date > datetime.datetime.now():
            time_to_wait = (self.test_start_date - datetime.datetime.now()).seconds
            logger.info(f"There are still {time_to_wait} seconds until test start")
            time.sleep(time_to_wait)
        self.logger.info(f"Start Time has reached current time, so starting the operation")

    @abstractmethod
    def stop(self):
        """
        Abstract method to implement the stop logic. This logic will be overwritten as per requirement for different
        runners.
        """
        pass

    @abstractmethod
    def run(self):
        """
        Abstract method to implement the logic for running a specific runner. This will be overwritten for
        different runners.
        """
        pass