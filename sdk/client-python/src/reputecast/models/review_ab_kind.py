from enum import StrEnum

class ReviewABKind(StrEnum):
    CONTENT = "content"
    SEND_TIME = "send_time"
    SUBJECT = "subject"

    def __str__(self) -> str:
        return str(self.value)
