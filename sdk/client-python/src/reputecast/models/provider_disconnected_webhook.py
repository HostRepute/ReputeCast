from enum import StrEnum

class ProviderDisconnectedWebhook(StrEnum):
    FAILED = "failed"
    MANUAL = "manual"
    NONE = "none"
    REMOVED = "removed"

    def __str__(self) -> str:
        return str(self.value)
