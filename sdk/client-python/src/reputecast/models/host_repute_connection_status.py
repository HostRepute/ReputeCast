from enum import StrEnum

class HostReputeConnectionStatus(StrEnum):
    CONNECTED = "connected"
    ERROR = "error"
    EXPIRED = "expired"
    NOT_CONNECTED = "not_connected"
    REVOKED = "revoked"

    def __str__(self) -> str:
        return str(self.value)
