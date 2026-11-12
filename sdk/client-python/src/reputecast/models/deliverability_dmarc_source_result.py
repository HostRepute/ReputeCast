from enum import StrEnum

class DeliverabilityDMARCSourceResult(StrEnum):
    FAIL = "fail"
    PASS = "pass"
    QUARANTINED = "quarantined"
    REJECTED = "rejected"

    def __str__(self) -> str:
        return str(self.value)
