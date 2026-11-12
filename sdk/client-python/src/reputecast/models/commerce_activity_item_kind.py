from enum import StrEnum

class CommerceActivityItemKind(StrEnum):
    CHECKOUT_ABANDONED = "checkout_abandoned"
    CHECKOUT_NOT_EMAILED = "checkout_not_emailed"
    CONNECTED = "connected"
    CUSTOMER_UNSUBSCRIBED = "customer_unsubscribed"
    DISCONNECTED = "disconnected"
    IMPORT_FINISHED = "import_finished"
    IMPORT_STARTED = "import_started"
    ORDER = "order"
    RATE_LIMITED = "rate_limited"
    RECONCILIATION = "reconciliation"
    RECONNECTED = "reconnected"
    REFUND = "refund"
    REVOKED = "revoked"
    SYNC_ERROR = "sync_error"
    UNSUBSCRIBE_PASSED = "unsubscribe_passed"
    WEBHOOK_MISSING = "webhook_missing"
    WEBHOOK_REGISTERED = "webhook_registered"

    def __str__(self) -> str:
        return str(self.value)
