from enum import StrEnum

class LiveMailboxStatus(StrEnum):
    DONE = "done"
    OK = "ok"
    PAUSED = "paused"
    SENDING = "sending"

    def __str__(self) -> str:
        return str(self.value)
