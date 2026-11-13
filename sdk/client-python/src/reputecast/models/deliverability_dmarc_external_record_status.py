from enum import StrEnum

class DeliverabilityDMARCExternalRecordStatus(StrEnum):
    FOUND = "found"
    MISSING = "missing"
    UNKNOWN = "unknown"

    def __str__(self) -> str:
        return str(self.value)
