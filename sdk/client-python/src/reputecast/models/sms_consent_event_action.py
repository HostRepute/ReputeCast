from enum import StrEnum

class SmsConsentEventAction(StrEnum):
    CONFIRMATION_FAILED = "confirmation_failed"
    CONFIRMATION_SENT = "confirmation_sent"
    CONFIRMED = "confirmed"
    IMPORTED = "imported"
    OPTED_IN = "opted_in"
    OPTED_OUT = "opted_out"
    PENDING = "pending"
    SUBSCRIBED = "subscribed"
    UNSUBSCRIBED = "unsubscribed"

    def __str__(self) -> str:
        return str(self.value)
