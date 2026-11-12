from enum import StrEnum

class ConnectTimelineStepStatus(StrEnum):
    DONE = "done"
    FAILED = "failed"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
