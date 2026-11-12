from enum import StrEnum

class ReportComparisonTone(StrEnum):
    BAD = "bad"
    GOOD = "good"
    NEUTRAL = "neutral"

    def __str__(self) -> str:
        return str(self.value)
