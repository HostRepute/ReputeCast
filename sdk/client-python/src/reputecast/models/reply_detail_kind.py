from enum import StrEnum

class ReplyDetailKind(StrEnum):
    OTHER = "other"
    REPLY = "reply"
    UNSUBSCRIBE = "unsubscribe"

    def __str__(self) -> str:
        return str(self.value)
