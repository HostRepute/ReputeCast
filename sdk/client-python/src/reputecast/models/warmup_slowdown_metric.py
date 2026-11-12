from enum import StrEnum

class WarmupSlowdownMetric(StrEnum):
    BOUNCE = "bounce"
    COMPLAINT = "complaint"

    def __str__(self) -> str:
        return str(self.value)
