from enum import StrEnum

class LeadAdsConnectionItemStatus(StrEnum):
    ATTENTION = "attention"
    CONNECTED = "connected"
    DISCONNECTED = "disconnected"

    def __str__(self) -> str:
        return str(self.value)
