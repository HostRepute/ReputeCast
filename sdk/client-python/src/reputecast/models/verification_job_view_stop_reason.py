from enum import StrEnum

class VerificationJobViewStopReason(StrEnum):
    CANCELLED = "cancelled"
    KEY_REJECTED = "key_rejected"
    NO_CREDITS = "no_credits"
    SERVICE_DOWN = "service_down"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
