from enum import StrEnum

class EarnEventStatus(StrEnum):
    APPLIED = "applied"
    FAILED = "failed"
    IGNORED = "ignored"
    PENDING = "pending"
    REJECTED = "rejected"

    def __str__(self) -> str:
        return str(self.value)
