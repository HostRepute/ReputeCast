from enum import StrEnum

class ImportJobStatus(StrEnum):
    CANCELED = "canceled"
    DONE = "done"
    FAILED = "failed"
    QUEUED = "queued"
    RUNNING = "running"

    def __str__(self) -> str:
        return str(self.value)
