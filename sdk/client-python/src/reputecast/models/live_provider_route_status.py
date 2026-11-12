from enum import StrEnum

class LiveProviderRouteStatus(StrEnum):
    CANCELLED = "cancelled"
    DONE = "done"
    HELD = "held"
    PAUSED = "paused"
    PENDING = "pending"
    SENDING = "sending"

    def __str__(self) -> str:
        return str(self.value)
