from enum import StrEnum

class ListWebhookDeliveriesState(StrEnum):
    DEAD = "dead"
    DELIVERED = "delivered"
    FAILED = "failed"
    HELD = "held"
    PENDING = "pending"
    RETRYING = "retrying"

    def __str__(self) -> str:
        return str(self.value)
