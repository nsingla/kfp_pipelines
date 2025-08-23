import os

class LoggingConfig:
    """Configuration class to handle logging"""
    LOG_DIRECTORY = os.getenv('log_directory', f'{os.getenv("HOME")}/logs/sqaa_logs')
    """str: log directory in which to write the log files"""
    LOG_FILENAME = os.getenv('log_file_name', 'wasr_test')
    """str: log filename prefix. a suffix will be added by the python logger as files are created."""
    """bool: switch to enable or disable http requests logging"""
    MAXIMUM_LOGFILE_BYTES = os.getenv('maximum_logfile_bytes', 5 * 1024 * 1024)
    """int: Number of rotating log files to create. Equivalent to python logging backupCount"""
    ROTATING_LOG_COUNT = os.getenv('rotating_log_count', 10)
    """int: Number of rotating log files to create. Equivalent to python logging backupCount"""
    LOG_LEVEL = os.getenv('log_level', "DEBUG")

