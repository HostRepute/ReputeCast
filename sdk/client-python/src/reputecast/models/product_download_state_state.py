from enum import StrEnum

class ProductDownloadStateState(StrEnum):
    DELIVERED = "delivered"
    EXPIRED = "expired"
    FAILED = "failed"
    PENDING = "pending"
    REVOKED = "revoked"
    REVOKED_AUTO = "revoked_auto"
    SENT = "sent"
    USED_UP = "used_up"

    def __str__(self) -> str:
        return str(self.value)
