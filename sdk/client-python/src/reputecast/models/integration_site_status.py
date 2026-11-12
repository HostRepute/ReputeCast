from enum import StrEnum

class IntegrationSiteStatus(StrEnum):
    ATTENTION = "attention"
    CONNECTED = "connected"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
