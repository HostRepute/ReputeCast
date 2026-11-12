from enum import StrEnum

class TransactionalSendResultStatus(StrEnum):
    NOT_DELIVERED = "not_delivered"
    QUEUED = "queued"
    SCHEDULED = "scheduled"

    def __str__(self) -> str:
        return str(self.value)
