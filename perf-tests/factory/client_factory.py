import kfp
from config.test_config import TestConfig

from logging import Logger
from logger import logger as test_logger


class ClientFactory:
    _instance = None
    logger: Logger = test_logger.Logger().logger
    kfp_client: kfp.Client = None

    def __new__(cls, *args, **kwargs):
        if cls._instance is None:
            cls._instance = super(ClientFactory, cls).__new__(cls)
        cls.setup()
        return cls._instance

    @classmethod
    def setup(cls):
        verify_ssl = False if TestConfig.LOCAL else True
        if TestConfig.IS_KUBEFLOW_MODE:
            cls.logger.info("Multi User Mode")
            cls.kfp_client = kfp.Client(host=TestConfig.KFP_url, namespace=TestConfig.NAMESPACE, verify_ssl=verify_ssl)
        else:
            cls.logger.info("Single User Mode")
            cls.kfp_client = kfp.Client(host=TestConfig.KFP_url, namespace=TestConfig.NAMESPACE, verify_ssl=verify_ssl)