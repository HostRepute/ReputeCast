from enum import StrEnum

class SenderDefaultsReplyHandling(StrEnum):
    FORWARD = "forward"
    INBOX = "inbox"

    def __str__(self) -> str:
        return str(self.value)
