from enum import StrEnum

class SenderDefaultsInputReplyHandling(StrEnum):
    FORWARD = "forward"

    def __str__(self) -> str:
        return str(self.value)
