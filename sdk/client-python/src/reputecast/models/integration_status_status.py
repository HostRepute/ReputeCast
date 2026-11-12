from enum import StrEnum

class IntegrationStatusStatus(StrEnum):
    ATTENTION = "attention"
    CONNECTED = "connected"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
