from enum import StrEnum

class TransactionalBatchItemStatus(StrEnum):
    ERROR = "error"
    NOT_DELIVERED = "not_delivered"
    QUEUED = "queued"
    SCHEDULED = "scheduled"

    def __str__(self) -> str:
        return str(self.value)
