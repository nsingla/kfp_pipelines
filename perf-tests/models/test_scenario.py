from typing import Optional, Any
from pydantic import BaseModel, Field, ConfigDict
from serde.enum_serde import EnumSerDe
from enums.test_mode import TestMode

class TestScenario(BaseModel):
    """
    Test Scenario class to read data from json file and convert the file into a test scenario object.
    """
    model_config = ConfigDict(extra='forbid', arbitrary_types_allowed=True, defer_build=True)
    mode: EnumSerDe(TestMode).enum_by_name()
    pipeline_file: Optional[str] = Field(alias="pipelineFileName", default=None)
    num_times: Optional[int] = Field(alias="numTimes", default=1)
    start_time: int = Field(alias="startTime")
    run_time: int = Field(alias="runTime")
    params: Optional[dict[str, Any]] = Field(alias="params", default=None)
    cron: Optional[str] = Field(default=None)