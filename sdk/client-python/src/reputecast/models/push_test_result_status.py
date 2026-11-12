from enum import StrEnum

class PushTestResultStatus(StrEnum):
    CLICKED = "clicked"
    FAILED = "failed"
    QUEUED = "queued"
    SENT = "sent"
    SHOWN = "shown"

    def __str__(self) -> str:
        return str(self.value)
