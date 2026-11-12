from enum import StrEnum

class ReportNotSentReadiness(StrEnum):
    BLOCKED = "blocked"
    READY = "ready"
    VALUE_3 = ""
    WARNINGS = "warnings"

    def __str__(self) -> str:
        return str(self.value)
