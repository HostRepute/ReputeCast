from enum import StrEnum

class TransactionalTimelineEntryKind(StrEnum):
    ACCEPTED = "accepted"
    BOUNCED = "bounced"
    CANCELLED = "cancelled"
    CLICKED = "clicked"
    COMPLAINED = "complained"
    DEFERRED = "deferred"
    DELIVERED = "delivered"
    FAILED = "failed"
    FAILED_OVER = "failed_over"
    HANDED = "handed"
    NOT_DELIVERED = "not_delivered"
    OPENED = "opened"
    PROVIDER_REJECTED = "provider_rejected"
    REJECTED = "rejected"
    RENDERED = "rendered"
    RETRY = "retry"
    ROUTED = "routed"
    SCHEDULED = "scheduled"
    WEBHOOK = "webhook"

    def __str__(self) -> str:
        return str(self.value)
