from typing import Optional, Any
from pydantic import BaseModel, Field, ConfigDict, model_validator
from serde.enum_serde import EnumSerDe
from enums.test_mode import TestMode

class TestScenario(BaseModel):
    """
    Test Scenario class to read data from json file and convert the file into a test scenario object.
    """
    model_config = ConfigDict(extra='forbid', arbitrary_types_allowed=True, defer_build=True)
    mode: EnumSerDe(TestMode).enum_by_name()
    pipeline_file: Optional[str] = Field(alias="pipelineFileName", default=None)
    num_times: Optional[int] = Field(alias="numTimes", default=1, description="Number of pipeline runs to execute or number of pipelines to upload")
    start_time: int = Field(alias="startTime")
    run_time: int = Field(alias="runTime",description="The time in minutes to run the mode")
    params: Optional[dict[str, Any]] = Field(alias="params", default=None)
    cron: Optional[str] = Field(default=None)

    @model_validator(mode='after')
    def validate_scenario(self) -> 'TestScenario':
        """
        Validates the test scenario based on mode and other constraints.
        """
        # Validate timing constraints
        if self.start_time < 0:
            raise ValueError(f"startTime must be non-negative, got {self.start_time}")
        
        if self.run_time <= 0:
            raise ValueError(f"runTime must be positive, got {self.run_time}")
        
        # Validate num_times constraints
        if self.num_times is not None and self.num_times <= 0:
            raise ValueError(f"numTimes must be positive, got {self.num_times}")
        
        # Mode-specific validations
        if self.mode == TestMode.PIPELINE_RUN or self.mode == TestMode.EXPERIMENT or self.mode == TestMode.PIPELINE_UPLOAD or self.mode == TestMode.PIPELINE_SCHEDULED_RUN:
            if not self.pipeline_file:
                raise ValueError("pipelineFileName is required for PIPELINE_RUN mode")
            
        elif self.mode == TestMode.PIPELINE_UPLOAD:
            if self.num_times is None:
                raise ValueError("numTimes is required for PIPELINE_UPLOAD mode")
        
        elif self.mode == TestMode.PIPELINE_SCHEDULED_RUN:
            if not self.cron:
                raise ValueError("cron expression is required for PIPELINE_SCHEDULED_RUN mode")
            if not self._is_valid_cron(self.cron):
                raise ValueError(f"Invalid cron expression format: {self.cron}")
        
        return self
    
    def _is_valid_cron(self, cron: str) -> bool:
        """
        Basic validation for cron expression format.
        Expects standard cron format: minute hour day month day_of_week
        """
        parts = cron.split()
        if len(parts) != 5:
            return False
        
        # Basic validation - could be enhanced with more sophisticated cron parsing
        try:
            minute, hour, day, month, day_of_week = parts
            
            # Validate ranges
            if not (0 <= int(minute) <= 59):
                return False
            if not (0 <= int(hour) <= 23):
                return False
            if not (1 <= int(day) <= 31):
                return False
            if not (1 <= int(month) <= 12):
                return False
            if not (0 <= int(day_of_week) <= 6):
                return False
                
        except (ValueError, TypeError):
            return False
        
        return True