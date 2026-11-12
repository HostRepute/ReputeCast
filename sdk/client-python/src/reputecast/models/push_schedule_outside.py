from enum import StrEnum

class PushScheduleOutside(StrEnum):
    HOLD = "hold"
    SKIP = "skip"

    def __str__(self) -> str:
        return str(self.value)
