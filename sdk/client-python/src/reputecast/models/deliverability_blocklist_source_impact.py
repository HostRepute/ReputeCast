from enum import StrEnum

class DeliverabilityBlocklistSourceImpact(StrEnum):
    HIGH = "high"
    LOW = "low"
    MEDIUM = "medium"

    def __str__(self) -> str:
        return str(self.value)
