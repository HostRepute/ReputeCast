from enum import StrEnum

class GrowthRetentionState(StrEnum):
    DONE = "done"
    IN_PROGRESS = "in_progress"
    NOT_STARTED = "not_started"

    def __str__(self) -> str:
        return str(self.value)
