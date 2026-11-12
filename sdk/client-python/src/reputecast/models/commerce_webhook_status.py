from enum import StrEnum

class CommerceWebhookStatus(StrEnum):
    FAILED = "failed"
    MISSING = "missing"
    PENDING = "pending"
    REGISTERED = "registered"

    def __str__(self) -> str:
        return str(self.value)
