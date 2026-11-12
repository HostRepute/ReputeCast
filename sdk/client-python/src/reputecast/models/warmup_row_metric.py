from enum import StrEnum

class WarmupRowMetric(StrEnum):
    BOUNCE = "bounce"
    COMPLAINT = "complaint"

    def __str__(self) -> str:
        return str(self.value)
