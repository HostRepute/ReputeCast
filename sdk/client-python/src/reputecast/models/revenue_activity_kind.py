from enum import StrEnum

class RevenueActivityKind(StrEnum):
    CANCEL_SCHEDULED = "cancel_scheduled"
    GIFT_REDEEMED = "gift_redeemed"
    OFFER_APPLIED = "offer_applied"
    PAYMENT_FAILED = "payment_failed"
    REFUNDED = "refunded"
    SUBSCRIBED = "subscribed"

    def __str__(self) -> str:
        return str(self.value)
