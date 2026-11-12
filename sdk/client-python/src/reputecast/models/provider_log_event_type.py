from enum import StrEnum

class ProviderLogEventType(StrEnum):
    COMPLAINT = "complaint"
    DEFERRED = "deferred"
    DELIVERED = "delivered"
    HARD_BOUNCE = "hard_bounce"
    REJECTED = "rejected"
    SOFT_BOUNCE = "soft_bounce"

    def __str__(self) -> str:
        return str(self.value)
