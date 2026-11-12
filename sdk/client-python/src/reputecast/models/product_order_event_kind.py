from enum import StrEnum

class ProductOrderEventKind(StrEnum):
    ACCESS_RESTORED = "access_restored"
    ACCESS_REVOKED = "access_revoked"
    BUYER_ASKED = "buyer_asked"
    CHECKOUT_STARTED = "checkout_started"
    DELIVERY_DELIVERED = "delivery_delivered"
    DELIVERY_FAILED = "delivery_failed"
    DELIVERY_SENT = "delivery_sent"
    DISPUTE_CLOSED = "dispute_closed"
    DISPUTE_OPENED = "dispute_opened"
    DISPUTE_UPDATED = "dispute_updated"
    DOWNLOADED = "downloaded"
    NEW_LINK = "new_link"
    PAID = "paid"
    REFUNDED = "refunded"

    def __str__(self) -> str:
        return str(self.value)
