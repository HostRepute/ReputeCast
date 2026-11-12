from enum import StrEnum

class LiveSnapshotState(StrEnum):
    CANCELLED = "cancelled"
    FAILED = "failed"
    FINISHED = "finished"
    IDLE = "idle"
    PARTIAL = "partial"
    PAUSED = "paused"
    PREPARING = "preparing"
    SCHEDULED = "scheduled"
    SENDING = "sending"

    def __str__(self) -> str:
        return str(self.value)
