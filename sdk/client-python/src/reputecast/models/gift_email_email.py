from enum import StrEnum

class GiftEmailEmail(StrEnum):
    GIFT_ENDING = "gift_ending"
    GIFT_PAYMENT_FAILED = "gift_payment_failed"
    GIFT_PURCHASED = "gift_purchased"
    GIFT_RECEIVED = "gift_received"

    def __str__(self) -> str:
        return str(self.value)
