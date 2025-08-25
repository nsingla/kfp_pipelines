import kfp
from config.test_config import TestConfig

from logging import Logger
from logger import logger

logger: Logger = logger.Logger().logger

class ClientFactory:

    kfp_client: kfp.client.client.Client = None

    def __init__(self):
        verify_ssl = False if TestConfig.LOCAL else True
        if TestConfig.IS_KUBEFLOW_MODE:
            logger.info("Multi User Mode")
            self.kfp_client = kfp.Client(host=TestConfig.KFP_url, namespace=TestConfig.NAMESPACE, verify_ssl=verify_ssl)
        else:
            logger.info("Single User Mode")
            self.kfp_client = kfp.Client(host=TestConfig.KFP_url, namespace=TestConfig.NAMESPACE, verify_ssl=verify_ssl)