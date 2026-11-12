from enum import StrEnum

class SenderVerificationStatus(StrEnum):
    FAILED = "failed"
    NOT_SET_UP = "not_set_up"
    NOT_USED = "not_used"
    PENDING = "pending"
    VERIFIED = "verified"

    def __str__(self) -> str:
        return str(self.value)
