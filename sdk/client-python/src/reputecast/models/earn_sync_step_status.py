from enum import StrEnum

class EarnSyncStepStatus(StrEnum):
    DONE = "done"
    FAILED = "failed"
    PENDING = "pending"

    def __str__(self) -> str:
        return str(self.value)
