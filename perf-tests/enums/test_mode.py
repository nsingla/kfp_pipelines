from enum import Enum, auto


class TestMode(Enum):
    PIPELINE_RUN = auto()
    RANDOM_GETS = auto()
    EXPERIMENT = auto()
    PIPELINE_UPLOAD = auto()
    PIPELINE_SCHEDULED_RUN = auto()