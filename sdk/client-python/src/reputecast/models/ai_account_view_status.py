from enum import StrEnum

class AIAccountViewStatus(StrEnum):
    CONNECTED = "connected"
    DISCONNECTED = "disconnected"
    KEY_REJECTED = "key_rejected"

    def __str__(self) -> str:
        return str(self.value)
