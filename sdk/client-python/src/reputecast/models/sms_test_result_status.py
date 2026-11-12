from enum import StrEnum

class SmsTestResultStatus(StrEnum):
    CANCELLED = "cancelled"
    DELIVERED = "delivered"
    FAILED = "failed"
    HELD = "held"
    QUEUED = "queued"
    SENDING = "sending"
    SENT = "sent"
    SKIPPED = "skipped"
    UNDELIVERED = "undelivered"

    def __str__(self) -> str:
        return str(self.value)
