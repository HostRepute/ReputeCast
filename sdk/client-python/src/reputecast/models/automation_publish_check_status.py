from enum import StrEnum

class AutomationPublishCheckStatus(StrEnum):
    BAD = "bad"
    OK = "ok"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
