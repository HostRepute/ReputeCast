from enum import StrEnum

class ImportDetailStatus(StrEnum):
    CANCELLED = "cancelled"
    DONE = "done"
    DRAFT = "draft"
    FAILED = "failed"
    MAPPING = "mapping"
    PAUSED = "paused"
    PREVIEWING = "previewing"
    RUNNING = "running"
    STAGING = "staging"

    def __str__(self) -> str:
        return str(self.value)
