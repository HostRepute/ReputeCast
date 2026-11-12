from enum import StrEnum

class WebhookViewStatus(StrEnum):
    FAILING = "failing"
    HEALTHY = "healthy"
    NEW = "new"
    PAUSED = "paused"

    def __str__(self) -> str:
        return str(self.value)
