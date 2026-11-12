from enum import StrEnum

class WhatsAppReviewOverLimit(StrEnum):
    NEXT_DAY = "next_day"
    SKIP = "skip"

    def __str__(self) -> str:
        return str(self.value)
