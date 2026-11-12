from enum import StrEnum

class ConnectConnectionEventsCheck(StrEnum):
    DEFERRED = "deferred"
    FAILED = "failed"
    NONE = "none"
    VERIFIED = "verified"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
