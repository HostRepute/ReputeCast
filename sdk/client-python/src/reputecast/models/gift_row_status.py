from enum import StrEnum

class GiftRowStatus(StrEnum):
    EXPIRED = "expired"
    NOT_REDEEMED = "not_redeemed"
    REDEEMED = "redeemed"
    REFUNDED = "refunded"
    SCHEDULED = "scheduled"

    def __str__(self) -> str:
        return str(self.value)
