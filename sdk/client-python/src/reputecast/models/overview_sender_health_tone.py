from enum import StrEnum

class OverviewSenderHealthTone(StrEnum):
    BAD = "bad"
    NEUTRAL = "neutral"
    OK = "ok"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
