from enum import StrEnum

class SmsPreviewPartKind(StrEnum):
    FIXED = "fixed"
    LINK = "link"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)
