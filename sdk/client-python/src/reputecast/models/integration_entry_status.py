from enum import StrEnum

class IntegrationEntryStatus(StrEnum):
    ATTENTION = "attention"
    CONNECTED = "connected"
    NONE = "none"
    NOT_CONNECTED = "not_connected"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
