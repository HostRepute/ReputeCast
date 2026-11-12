from enum import StrEnum

class PushScheduleMode(StrEnum):
    AT = "at"
    NOW = "now"

    def __str__(self) -> str:
        return str(self.value)
