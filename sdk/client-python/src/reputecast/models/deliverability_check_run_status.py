from enum import StrEnum

class DeliverabilityCheckRunStatus(StrEnum):
    DONE = "done"
    FAILED = "failed"
    RUNNING = "running"

    def __str__(self) -> str:
        return str(self.value)
