from enum import StrEnum

class DigestFeedItemStatus(StrEnum):
    BEFORE_START = "before_start"
    LEFT_OUT = "left_out"
    NEW = "new"
    SENT = "sent"

    def __str__(self) -> str:
        return str(self.value)
