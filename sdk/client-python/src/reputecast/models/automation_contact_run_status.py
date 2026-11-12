from enum import StrEnum

class AutomationContactRunStatus(StrEnum):
    ACTIVE = "active"
    COMPLETED = "completed"
    EXITED = "exited"
    FAILED = "failed"
    GOAL = "goal"

    def __str__(self) -> str:
        return str(self.value)
