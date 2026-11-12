from enum import StrEnum

class SendingDomainTrackingStatus(StrEnum):
    ERROR = "error"
    MISSING = "missing"
    NONE = "none"
    PENDING = "pending"
    REFUSED = "refused"
    VERIFIED = "verified"
    WRONG_VALUE = "wrong_value"

    def __str__(self) -> str:
        return str(self.value)
