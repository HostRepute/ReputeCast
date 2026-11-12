from enum import StrEnum

class SmsZoneStepStatus(StrEnum):
    HELD = "held"
    PAUSED = "paused"
    SENDING = "sending"
    SENT = "sent"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
