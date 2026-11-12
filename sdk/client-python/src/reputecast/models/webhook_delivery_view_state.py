from enum import StrEnum

class WebhookDeliveryViewState(StrEnum):
    DEAD = "dead"
    DELIVERED = "delivered"
    DELIVERING = "delivering"
    HELD = "held"
    PENDING = "pending"
    RETRYING = "retrying"

    def __str__(self) -> str:
        return str(self.value)
