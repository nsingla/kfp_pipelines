from typing import Annotated

from pydantic import BeforeValidator


class EnumSerDe:
    enum_type = None

    def __init__(self, enum_type):
        self.enum_type = enum_type

    def enum_by_name(self):
        return Annotated[self.enum_type, BeforeValidator(self._get_enum_by_name)]

    def _get_enum_by_name(self, v: str):
        if v is not None:
            try:
                return self.enum_type[v]
            except KeyError:
                raise ValueError(f"invalid value: {v}, allowed values: {self.enum_type._member_names_}")
        return None

    def json_encoder_enum_by_name(self):
        return {self.enum_type: lambda e: e.name}
