from enum import StrEnum

class EarnConnectionStatus(StrEnum):
    CONNECTED = "connected"
    SETUP = "setup"

    def __str__(self) -> str:
        return str(self.value)
