from enum import StrEnum

class UploadSuppressionsReason(StrEnum):
    COMPLAINT = "complaint"
    HARD_BOUNCE = "hard_bounce"
    MANUAL = "manual"
    UNSUBSCRIBE = "unsubscribe"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)
