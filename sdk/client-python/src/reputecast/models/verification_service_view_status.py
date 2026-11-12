from enum import StrEnum

class VerificationServiceViewStatus(StrEnum):
    CONNECTED = "connected"
    ERROR = "error"
    LOW_CREDITS = "low_credits"
    NOT_CONNECTED = "not_connected"

    def __str__(self) -> str:
        return str(self.value)
