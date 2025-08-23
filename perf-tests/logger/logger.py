import logging
import os
import time
from logging import handlers
from logging.handlers import RotatingFileHandler
from typing import ClassVar
from typing import Optional
from colorlog import ColoredFormatter
from config.logging_config import LoggingConfig


class Logger:
    """Main Logging Class to instantiate logger.

    Configures:
    - Color console handler
    - Rotation file handler
    - JSON handler
    - File handler
    """

    LOG_DIRECTORY = LoggingConfig.LOG_DIRECTORY
    LOG_FILE_NAME = LoggingConfig.LOG_FILENAME
    REQUESTS_LOG_LEVEL = 15
    LOG_LEVEL = LoggingConfig.LOG_LEVEL
    LOG_FORMAT = "%(log_color)s%(asctime)s | %(log_color)s%(levelname)-2s%(reset)s | %(log_color)s%(message)s%(reset)s"
    LOG_WITHOUT_COLOR_FORMAT = "%(asctime)s | %(levelname)-2s | %(message)s"
    LOG_COLORS = {
        "INFO": "green",
        "REQUESTS": "bold_white",
        "DEBUG": "light_white",
        "WARNING": "yellow",
        "ERROR": "red",
        "CRITICAL": "red"
    }

    # Current and the only instance of the class.
    instance: ClassVar[Optional["Logger"]]
    logger: ClassVar[logging.Logger]
    file_handler: ClassVar[handlers.RotatingFileHandler | None] = None

    def __new__(cls, new_instance: bool = False):
        """
        Set up a logger that writes to a file.
        A RotatingFileHandler is used to imit the size of the log file.

        :param new_instance: To configure a new handler
        """
        if not hasattr(cls, "instance"):
            logging.addLevelName(cls.REQUESTS_LOG_LEVEL, "REQUESTS")
            cls.instance = super().__new__(cls)
            cls.logger = logging.getLogger(__name__)
            cls._configure_logger(cls.logger)

        if new_instance:
            cls.initialize_file_handler()
        return cls.instance

    @classmethod
    def initialize_file_handler(cls) -> None:
        if cls.file_handler:
            cls.logger.warning(
                "The previous handler wasn't closed; closing it now before configuring a new one"
            )
            cls.shutdown_handler()
        cls.file_handler = cls._configure_file_handler(
            cls.logger,
            filename=f"{cls.LOG_FILE_NAME}-{int(time.time() * 100000)}",
            use_colors=False,
        )

    @classmethod
    def _configure_logger(cls, logger: logging.Logger) -> logging.Logger:
        """Configure main logger and it's handlers."""
        logger.propagate = False
        logger.setLevel(LoggingConfig.LOG_LEVEL)
        cls._configure_file_handler(logger, filename=cls.LOG_FILE_NAME)
        cls._configure_console_handler(logger)
        return logger

    @classmethod
    def _configure_file_handler(
        cls,
        logger: logging.Logger,
        filename: str,
        use_colors: bool = True,
    ) -> RotatingFileHandler:
        """Configure and attach rotation file handler to the given logger."""
        os.makedirs(cls.LOG_DIRECTORY, exist_ok=True)
        log_file = os.path.join(
            cls.LOG_DIRECTORY, f"{filename}.log" if not filename.endswith(".log") else filename
        )
        file_handler = handlers.RotatingFileHandler(
            filename=log_file,
            mode="w",
            maxBytes=LoggingConfig.MAXIMUM_LOGFILE_BYTES,
            backupCount=LoggingConfig.ROTATING_LOG_COUNT,
            encoding=None,
            delay=False,
        )
        if use_colors:
            log_formatter = ColoredFormatter(cls.LOG_FORMAT, log_colors=cls.LOG_COLORS)
        else:
            log_formatter = logging.Formatter(cls.LOG_WITHOUT_COLOR_FORMAT)
        file_handler.setFormatter(log_formatter)
        file_handler.setLevel(LoggingConfig.LOG_LEVEL)
        logger.addHandler(file_handler)
        return file_handler

    @classmethod
    def _configure_console_handler(cls, logger: logging.Logger, use_colors: bool = True) -> None:
        """Configure and attach console handler to the given logger."""
        console_handler = logging.StreamHandler()
        console_handler.setLevel(LoggingConfig.LOG_LEVEL)
        if use_colors:
            log_formatter = ColoredFormatter(cls.LOG_FORMAT, log_colors=cls.LOG_COLORS)
        else:
            log_formatter = logging.Formatter(cls.LOG_FORMAT)
        console_handler.setFormatter(log_formatter)
        logger.addHandler(console_handler)

    @classmethod
    def shutdown_handler(cls) -> None:
        """Gracefully close file handler."""
        if cls.file_handler:
            cls.file_handler.close()
            cls.logger.removeHandler(cls.file_handler)
        cls.file_handler = None

    def __hash__(self) -> int:
        return hash(time.time() * 100000)

    def get_log_file(self) -> str | None:
        for handler in self.logger.handlers:
            if isinstance(handler, logging.FileHandler):
                return handler.baseFilename
        return None