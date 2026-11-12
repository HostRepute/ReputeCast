from enum import StrEnum

class WhatsAppSendResultStatus(StrEnum):
    SCHEDULED = "scheduled"
    SENDING = "sending"

    def __str__(self) -> str:
        return str(self.value)
