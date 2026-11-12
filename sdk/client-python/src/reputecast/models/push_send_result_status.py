from enum import StrEnum

class PushSendResultStatus(StrEnum):
    SCHEDULED = "scheduled"
    SENDING = "sending"

    def __str__(self) -> str:
        return str(self.value)
