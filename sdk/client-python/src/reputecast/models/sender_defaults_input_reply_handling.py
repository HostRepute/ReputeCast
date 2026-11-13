from enum import StrEnum

class SenderDefaultsInputReplyHandling(StrEnum):
    FORWARD = "forward"
    INBOX = "inbox"

    def __str__(self) -> str:
        return str(self.value)
