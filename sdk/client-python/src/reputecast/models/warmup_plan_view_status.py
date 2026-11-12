from enum import StrEnum

class WarmupPlanViewStatus(StrEnum):
    ACTIVE = "active"
    DONE = "done"
    PAUSED = "paused"
    SLOWED = "slowed"

    def __str__(self) -> str:
        return str(self.value)
