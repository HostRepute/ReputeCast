from enum import StrEnum

class LeadAdsConnectionStatus(StrEnum):
    ATTENTION = "attention"
    CONNECTED = "connected"
    DISCONNECTED = "disconnected"

    def __str__(self) -> str:
        return str(self.value)
