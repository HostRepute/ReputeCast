from enum import StrEnum

class ReplyListItemStatus(StrEnum):
    DONE = "done"
    OPEN = "open"

    def __str__(self) -> str:
        return str(self.value)
