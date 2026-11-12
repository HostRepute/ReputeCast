from enum import StrEnum

class DeliverabilityReputationStatus(StrEnum):
    BLOCK = "block"
    PASS = "pass"
    UNKNOWN = "unknown"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
