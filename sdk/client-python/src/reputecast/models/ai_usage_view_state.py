from enum import StrEnum

class AIUsageViewState(StrEnum):
    NOTICE = "notice"
    OK = "ok"
    REACHED = "reached"

    def __str__(self) -> str:
        return str(self.value)
