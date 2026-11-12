from enum import StrEnum

class RemovalItemState(StrEnum):
    DRAFT = "draft"
    SCHEDULED = "scheduled"
    SENDING = "sending"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
