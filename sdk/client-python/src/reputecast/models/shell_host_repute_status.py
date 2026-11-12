from enum import StrEnum

class ShellHostReputeStatus(StrEnum):
    CONNECTED = "connected"
    ERROR = "error"
    EXPIRED = "expired"
    LIMIT_REACHED = "limit_reached"
    REVOKED = "revoked"

    def __str__(self) -> str:
        return str(self.value)
