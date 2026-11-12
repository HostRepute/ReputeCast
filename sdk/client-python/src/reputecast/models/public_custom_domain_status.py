from enum import StrEnum

class PublicCustomDomainStatus(StrEnum):
    FAILED = "failed"
    NONE = "none"
    PENDING = "pending"
    VERIFIED = "verified"

    def __str__(self) -> str:
        return str(self.value)
