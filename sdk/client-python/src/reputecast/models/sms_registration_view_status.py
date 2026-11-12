from enum import StrEnum

class SmsRegistrationViewStatus(StrEnum):
    ALLOWED = "allowed"
    APPROVED = "approved"
    NOT_REGISTERED = "not_registered"
    NOT_SUPPORTED = "not_supported"
    PENDING = "pending"
    REJECTED = "rejected"
    VERIFIED = "verified"

    def __str__(self) -> str:
        return str(self.value)
