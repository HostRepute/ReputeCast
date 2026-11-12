from enum import StrEnum

class VerificationJobViewStatus(StrEnum):
    CANCELLED = "cancelled"
    FINISHED = "finished"
    PAUSED = "paused"
    QUEUED = "queued"
    RUNNING = "running"
    STOPPED = "stopped"

    def __str__(self) -> str:
        return str(self.value)
