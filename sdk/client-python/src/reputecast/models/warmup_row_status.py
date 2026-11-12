from enum import StrEnum

class WarmupRowStatus(StrEnum):
    COUNTED = "counted"
    NOT_COUNTED = "not_counted"
    PLANNED = "planned"
    TODAY = "today"

    def __str__(self) -> str:
        return str(self.value)
