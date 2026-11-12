from enum import StrEnum

class SmsCatalogItemWebhooks(StrEnum):
    MANUAL = "manual"
    REGISTERED = "registered"

    def __str__(self) -> str:
        return str(self.value)
