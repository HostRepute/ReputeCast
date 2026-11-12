from enum import StrEnum

class ReferralItemStatus(StrEnum):
    AWAITING = "awaiting"
    COUNTED = "counted"
    HOLD = "hold"
    NOT_COUNTED = "not_counted"
    REVIEW = "review"

    def __str__(self) -> str:
        return str(self.value)
