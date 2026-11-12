from enum import StrEnum

class ConnectProvisionRowStatus(StrEnum):
    DEFERRED = "deferred"
    DONE = "done"
    FAILED = "failed"
    RUNNING = "running"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
