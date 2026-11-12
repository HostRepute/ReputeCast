from enum import StrEnum

class ConnectConnectionStatus(StrEnum):
    CONNECTED = "connected"
    FAILED = "failed"
    NONE = "none"

    def __str__(self) -> str:
        return str(self.value)
