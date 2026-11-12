from enum import StrEnum

class AIConnectionViewStatus(StrEnum):
    CONNECTED = "connected"
    KEY_REJECTED = "key_rejected"

    def __str__(self) -> str:
        return str(self.value)
