from enum import StrEnum

class TransactionalScheduleStatus(StrEnum):
    CANCELLED = "cancelled"
    SCHEDULED = "scheduled"

    def __str__(self) -> str:
        return str(self.value)
