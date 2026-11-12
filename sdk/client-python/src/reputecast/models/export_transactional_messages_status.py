from enum import StrEnum

class ExportTransactionalMessagesStatus(StrEnum):
    ALL = "all"
    BOUNCED = "bounced"
    CANCELLED = "cancelled"
    COMPLAINED = "complained"
    DEFERRED = "deferred"
    DELIVERED = "delivered"
    FAILED = "failed"
    QUEUED = "queued"
    REJECTED = "rejected"
    SCHEDULED = "scheduled"
    SENT = "sent"
    TEST = "test"

    def __str__(self) -> str:
        return str(self.value)
