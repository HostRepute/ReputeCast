from enum import StrEnum

class SenderDefaultsReplyHandling(StrEnum):
    FORWARD = "forward"

    def __str__(self) -> str:
        return str(self.value)
