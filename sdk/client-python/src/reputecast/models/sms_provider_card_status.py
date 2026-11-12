from enum import StrEnum

class SmsProviderCardStatus(StrEnum):
    CREDENTIALS_REJECTED = "credentials_rejected"
    DEGRADED = "degraded"
    DOWN = "down"
    HEALTHY = "healthy"

    def __str__(self) -> str:
        return str(self.value)
