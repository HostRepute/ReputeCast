from enum import StrEnum

class EarnMemberEmailEmail(StrEnum):
    CANCELED = "canceled"
    GIFT_ENDING = "gift_ending"
    GIFT_PAYMENT_FAILED = "gift_payment_failed"
    GIFT_PURCHASED = "gift_purchased"
    GIFT_RECEIVED = "gift_received"
    PAYMENT_FAILED = "payment_failed"
    RENEWAL_REMINDER = "renewal_reminder"
    TRIAL_ENDING = "trial_ending"
    WELCOME = "welcome"

    def __str__(self) -> str:
        return str(self.value)
