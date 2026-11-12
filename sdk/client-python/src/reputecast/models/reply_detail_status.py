from enum import StrEnum

class ReplyDetailStatus(StrEnum):
    DONE = "done"
    OPEN = "open"

    def __str__(self) -> str:
        return str(self.value)
