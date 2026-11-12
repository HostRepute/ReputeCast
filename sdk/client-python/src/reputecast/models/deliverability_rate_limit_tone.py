from enum import StrEnum

class DeliverabilityRateLimitTone(StrEnum):
    BAD = "bad"
    OK = "ok"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
