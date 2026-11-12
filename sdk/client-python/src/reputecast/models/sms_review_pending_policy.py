from enum import StrEnum

class SmsReviewPendingPolicy(StrEnum):
    SKIP = "skip"
    WAIT = "wait"

    def __str__(self) -> str:
        return str(self.value)
