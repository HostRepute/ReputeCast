from enum import StrEnum

class AutomationGraphIssueSeverity(StrEnum):
    ERROR = "error"
    WARNING = "warning"

    def __str__(self) -> str:
        return str(self.value)
