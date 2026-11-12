from enum import StrEnum

class SmsProviderCardWebhooks(StrEnum):
    MANUAL = "manual"
    REGISTERED = "registered"

    def __str__(self) -> str:
        return str(self.value)
