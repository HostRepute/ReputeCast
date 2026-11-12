from enum import StrEnum

class IntegrationWordPressPageStatus(StrEnum):
    ATTENTION = "attention"
    CONNECTED = "connected"
    NOT_CONNECTED = "not_connected"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
