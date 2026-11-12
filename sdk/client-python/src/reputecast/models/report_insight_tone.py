from enum import StrEnum

class ReportInsightTone(StrEnum):
    BAD = "bad"
    INFO = "info"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
