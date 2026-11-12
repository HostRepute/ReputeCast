from enum import StrEnum

class SendingDomainRowStatus(StrEnum):
    FAILED = "failed"
    PARTIAL = "partial"
    PENDING = "pending"
    VERIFIED = "verified"

    def __str__(self) -> str:
        return str(self.value)
