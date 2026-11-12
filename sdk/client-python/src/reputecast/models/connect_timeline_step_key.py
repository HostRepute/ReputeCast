from enum import StrEnum

class ConnectTimelineStepKey(StrEnum):
    ACCEPTED = "accepted"
    DELIVERED = "delivered"
    OPENED = "opened"
    QUEUED = "queued"
    SENT = "sent"

    def __str__(self) -> str:
        return str(self.value)
