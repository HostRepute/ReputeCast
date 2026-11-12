from enum import StrEnum

class WhatsAppSendStatePhase(StrEnum):
    CANCELLED = "cancelled"
    COMPLETED = "completed"
    SCHEDULED = "scheduled"
    SENDING = "sending"

    def __str__(self) -> str:
        return str(self.value)
