from enum import StrEnum

class ReportBounceRowTone(StrEnum):
    BAD = "bad"
    NEUTRAL = "neutral"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
