from enum import StrEnum

class SmsSendViewPhase(StrEnum):
    CANCELLED = "cancelled"
    COMPLETED = "completed"
    PAUSED = "paused"
    SCHEDULED = "scheduled"
    SENDING = "sending"

    def __str__(self) -> str:
        return str(self.value)
