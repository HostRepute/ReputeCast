from enum import StrEnum

class SendingDomainRecordStatus(StrEnum):
    ERROR = "error"
    MISSING = "missing"
    PENDING = "pending"
    VERIFIED = "verified"
    WRONG_VALUE = "wrong_value"

    def __str__(self) -> str:
        return str(self.value)
