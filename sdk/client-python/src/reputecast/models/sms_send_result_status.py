from enum import StrEnum

class SmsSendResultStatus(StrEnum):
    SCHEDULED = "scheduled"
    SENDING = "sending"

    def __str__(self) -> str:
        return str(self.value)
