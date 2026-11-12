from enum import StrEnum

class ComposerDismissalReason(StrEnum):
    DECORATIVE = "decorative"
    OTHER = "other"
    SIZE = "size"
    SUBJECT = "subject"
    WRONG = "wrong"

    def __str__(self) -> str:
        return str(self.value)
