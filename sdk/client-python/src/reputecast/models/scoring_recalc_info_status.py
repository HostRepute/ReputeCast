from enum import StrEnum

class ScoringRecalcInfoStatus(StrEnum):
    FAILED = "failed"
    FINISHED = "finished"
    RUNNING = "running"
    SUPERSEDED = "superseded"

    def __str__(self) -> str:
        return str(self.value)
