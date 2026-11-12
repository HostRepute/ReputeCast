from enum import StrEnum

class SegmentConditionUnit(StrEnum):
    DAYS = "days"
    WEEKS = "weeks"

    def __str__(self) -> str:
        return str(self.value)
